package ipam

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strings"

	"github.com/avast/retry-go/v4"
	"go4.org/netipx"
)

// Prefix is a expression of a ip with length and forms a classless network.
type Prefix struct {
	Cidr                   string          `json:"Cidr"`       // The Cidr of this prefix
	ParentCidr             string          `json:"ParentCidr"` // if this prefix is a child this is a pointer back
	isParent               bool            // if this Prefix has child prefixes, this is set to true
	availableChildPrefixes map[string]bool // available child prefixes of this prefix
	// TODO remove this in the next release
	childPrefixLength int             // the length of the child prefixes
	ips               map[string]bool // The ips contained in this prefix
	version           int64           // version is used for optimistic locking
}

type Prefixes []Prefix

// deepCopy to a new Prefix
func (p *Prefix) deepCopy() *Prefix {
	return &Prefix{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		isParent:               p.isParent,
		childPrefixLength:      p.childPrefixLength,
		availableChildPrefixes: copyMap(p.availableChildPrefixes),
		ips:                    copyMap(p.ips),
		version:                p.version,
	}
}

// GobEncode implements GobEncode for Prefix
func (p *Prefix) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	if err := encoder.Encode(p.availableChildPrefixes); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.childPrefixLength); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.isParent); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.ips); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.version); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.Cidr); err != nil {
		return nil, err
	}
	if err := encoder.Encode(p.ParentCidr); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// GobDecode implements GobDecode for Prefix
func (p *Prefix) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	if err := decoder.Decode(&p.availableChildPrefixes); err != nil {
		return err
	}
	if err := decoder.Decode(&p.childPrefixLength); err != nil {
		return err
	}
	if err := decoder.Decode(&p.isParent); err != nil {
		return err
	}
	if err := decoder.Decode(&p.ips); err != nil {
		return err
	}
	if err := decoder.Decode(&p.version); err != nil {
		return err
	}
	if err := decoder.Decode(&p.Cidr); err != nil {
		return err
	}
	return decoder.Decode(&p.ParentCidr)
}

// TODO replace with maps.Copy
func copyMap(m map[string]bool) map[string]bool {
	cm := make(map[string]bool, len(m))
	for k, v := range m {
		cm[k] = v
	}
	return cm
}

// Usage of ips and child Prefixes of a Prefix
type Usage struct {
	// AvailableIPs the number of available IPs if this is not a parent prefix
	// No more than 2^31 available IPs are reported
	AvailableIPs uint64
	// AcquiredIPs the number of acquired IPs if this is not a parent prefix
	AcquiredIPs uint64
	// AvailableSmallestPrefixes is the count of available Prefixes with 2 countable Bits
	// No more than 2^31 available Prefixes are reported
	AvailableSmallestPrefixes uint64
	// AvailablePrefixes is a list of prefixes which are available
	AvailablePrefixes []string
	// AcquiredPrefixes the number of acquired prefixes if this is a parent prefix
	AcquiredPrefixes uint64
}

func (i *ipamer) NewPrefix(ctx context.Context, cidr string) (*Prefix, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	namespace := namespaceFromContext(ctx)
	existingPrefixes, err := i.storage.ReadAllPrefixCidrs(ctx, namespace)
	if err != nil {
		return nil, err
	}
	p, err := i.newPrefix(cidr, "")
	if err != nil {
		return nil, err
	}
	err = PrefixesOverlapping(existingPrefixes, []string{p.Cidr})
	if err != nil {
		return nil, err
	}
	newPrefix, err := i.storage.CreatePrefix(ctx, *p, namespace)
	if err != nil {
		return nil, err
	}

	return &newPrefix, nil
}

func (i *ipamer) DeletePrefix(ctx context.Context, cidr string) (*Prefix, error) {
	namespace := namespaceFromContext(ctx)
	p, err := i.PrefixFrom(ctx, cidr)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to find prefix for cidr:%s error:%s", ErrNotFound, cidr, err.Error())
	}
	if p == nil {
		return nil, fmt.Errorf("%w: delete prefix:%s", ErrNotFound, cidr)
	}
	if p.hasIPs() {
		return nil, fmt.Errorf("prefix %s has ips, delete prefix not possible", p.Cidr)
	}
	prefix, err := i.storage.DeletePrefix(ctx, *p, namespace)
	if err != nil {
		return nil, fmt.Errorf("delete prefix:%s %w", cidr, err)
	}

	return &prefix, nil
}

func (i *ipamer) AcquireChildPrefix(ctx context.Context, parentCidr string, length uint8) (*Prefix, error) {
	namespace := namespaceFromContext(ctx)
	var prefix *Prefix
	return prefix, retryOnOptimisticLock(func() error {
		var err error
		prefix, err = i.acquireChildPrefixInternal(ctx, namespace, parentCidr, "", int(length))
		return err
	})
}

func (i *ipamer) AcquireSpecificChildPrefix(ctx context.Context, parentCidr, childCidr string) (*Prefix, error) {
	namespace := namespaceFromContext(ctx)
	var prefix *Prefix
	return prefix, retryOnOptimisticLock(func() error {
		var err error
		prefix, err = i.acquireChildPrefixInternal(ctx, namespace, parentCidr, childCidr, 0)
		return err
	})
}

// acquireChildPrefixInternal will return a Prefix with a smaller length from the given Prefix.
func (i *ipamer) acquireChildPrefixInternal(ctx context.Context, namespace, parentCidr, childCidr string, length int) (*Prefix, error) {
	specificChildRequest := childCidr != ""
	var childprefix netip.Prefix
	parent, err := i.PrefixFrom(ctx, parentCidr)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to find prefix for cidr:%s error:%s", ErrNotFound, parentCidr, err.Error())
	}
	if parent == nil {
		return nil, fmt.Errorf("unable to find prefix for cidr:%s", parentCidr)
	}
	ipprefix, err := netip.ParsePrefix(parent.Cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse parent.cidr:%s of parentCidr:%s %w", parent.Cidr, parentCidr, err)
	}
	if specificChildRequest {
		childprefix, err = netip.ParsePrefix(childCidr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse childCidr:%s %w", childCidr, err)
		}
		length = childprefix.Bits()
	}
	if ipprefix.Bits() >= length {
		return nil, fmt.Errorf("given length:%d must be greater than prefix length:%d", length, ipprefix.Bits())
	}
	if parent.hasIPs() {
		return nil, fmt.Errorf("prefix %s has ips, acquire child prefix not possible", parent.Cidr)
	}

	var ipsetBuilder netipx.IPSetBuilder
	ipsetBuilder.AddPrefix(ipprefix)
	for cp, available := range parent.availableChildPrefixes {
		if available {
			continue
		}
		cpipprefix, err := netip.ParsePrefix(cp)
		if err != nil {
			return nil, err
		}
		ipsetBuilder.RemovePrefix(cpipprefix)
	}

	ipset, err := ipsetBuilder.IPSet()
	if err != nil {
		return nil, fmt.Errorf("error constructing ipset:%w", err)
	}

	var cp netip.Prefix
	if !specificChildRequest {
		var ok bool
		cp, _, ok = ipset.RemoveFreePrefix(uint8(length)) // nolint:gosec
		if !ok {
			pfxs := ipset.Prefixes()
			if len(pfxs) == 0 {
				return nil, fmt.Errorf("no prefix found in %s with length:%d", parentCidr, length)
			}

			var availablePrefixes []string
			for _, p := range pfxs {
				availablePrefixes = append(availablePrefixes, p.String())
			}
			adj := "are"
			if len(availablePrefixes) == 1 {
				adj = "is"
			}

			return nil, fmt.Errorf("no prefix found in %s with length:%d, but %s %s available", parentCidr, length, strings.Join(availablePrefixes, ","), adj)
		}
	} else {
		if ok := ipset.ContainsPrefix(childprefix); !ok {
			// Parent prefix does not contain specific child prefix
			return nil, fmt.Errorf("specific prefix %s is not available in prefix %s", childCidr, parentCidr)
		}
		cp = childprefix
	}

	// Ensure acquired child prefix is valid
	if !cp.IsValid() {
		return nil, fmt.Errorf("acquired child prefix:%s is not valid", cp.String())
	}

	child := &Prefix{
		Cidr:       cp.String(),
		ParentCidr: parentCidr,
	}

	parent.availableChildPrefixes[child.Cidr] = false
	parent.isParent = true

	_, err = i.storage.UpdatePrefix(ctx, *parent, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to update parent prefix:%v error:%w", parent, err)
	}
	child, err = i.newPrefix(child.Cidr, parentCidr)
	if err != nil {
		return nil, fmt.Errorf("unable to persist created child:%w", err)
	}
	_, err = i.storage.CreatePrefix(ctx, *child, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to update parent prefix:%v error:%w", child, err)
	}

	return child, nil
}

func (i *ipamer) ReleaseChildPrefix(ctx context.Context, child *Prefix) error {
	namespace := namespaceFromContext(ctx)
	return retryOnOptimisticLock(func() error {
		return i.releaseChildPrefixInternal(ctx, namespace, child)
	})
}

// releaseChildPrefixInternal will mark this child Prefix as available again.
func (i *ipamer) releaseChildPrefixInternal(ctx context.Context, namespace string, child *Prefix) error {
	parent, err := i.PrefixFrom(ctx, child.ParentCidr)
	if err != nil {
		return fmt.Errorf("%w: unable to find prefix for cidr:%q error:%s", ErrNotFound, child.ParentCidr, err.Error())
	}
	if parent == nil || !parent.isParent {
		return fmt.Errorf("prefix:%q is no child prefix", child.Cidr)
	}
	if len(child.ips) > 2 {
		return fmt.Errorf("prefix %s has ips, deletion not possible", child.Cidr)
	}

	parent.availableChildPrefixes[child.Cidr] = true
	_, err = i.storage.UpdatePrefix(ctx, *parent, namespace)
	if err != nil {
		return fmt.Errorf("unable to update parent:%q to release child prefix:%q :%w", child.ParentCidr, child.Cidr, err)
	}

	_, err = i.DeletePrefix(ctx, child.Cidr)
	if err != nil {
		return fmt.Errorf("unable to delete child prefix:%q :%w", child.Cidr, err)
	}

	return nil
}

func (i *ipamer) PrefixFrom(ctx context.Context, cidr string) (*Prefix, error) {
	namespace := namespaceFromContext(ctx)
	ipprefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}
	prefix, err := i.storage.ReadPrefix(ctx, ipprefix.Masked().String(), namespace)
	if err != nil {
		return nil, err
	}
	return &prefix, nil
}

func (i *ipamer) AcquireSpecificIP(ctx context.Context, prefixCidr, specificIP string) (*IP, error) {
	namespace := namespaceFromContext(ctx)
	var ip *IP
	return ip, retryOnOptimisticLock(func() error {
		var err error
		ip, err = i.acquireSpecificIPInternal(ctx, namespace, prefixCidr, specificIP)
		return err
	})
}

// acquireSpecificIPInternal will acquire given IP and mark this IP as used, if already in use, return nil.
// If specificIP is empty, the next free IP is returned.
// If there is no free IP an NoIPAvailableError is returned.
// If the Prefix is not found an NotFoundError is returned.
func (i *ipamer) acquireSpecificIPInternal(ctx context.Context, namespace, prefixCidr, specificIP string) (*IP, error) {
	prefix, err := i.PrefixFrom(ctx, prefixCidr)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to find prefix for cidr:%s error:%s", ErrNotFound, prefixCidr, err.Error())
	}
	if prefix == nil {
		return nil, fmt.Errorf("%w: unable to find prefix for cidr:%s", ErrNotFound, prefixCidr)
	}
	if prefix.isParent {
		return nil, fmt.Errorf("prefix %s has childprefixes, acquire ip not possible", prefix.Cidr)
	}
	ipnet, err := netip.ParsePrefix(prefix.Cidr)
	if err != nil {
		return nil, err
	}

	var specificIPnet netip.Addr
	if specificIP != "" {
		specificIPnet, err = netip.ParseAddr(specificIP)
		if err != nil {
			return nil, fmt.Errorf("given ip:%s in not valid", specificIP)
		}
		if !ipnet.Contains(specificIPnet) {
			return nil, fmt.Errorf("given ip:%s is not in %s", specificIP, prefixCidr)
		}
		_, ok := prefix.ips[specificIPnet.String()]
		if ok {
			return nil, fmt.Errorf("%w: given ip:%s is already allocated", ErrAlreadyAllocated, specificIPnet)
		}
		return i.acquireAndStore(ctx, namespace, prefix, specificIPnet)
	}

	iprange := netipx.RangeOfPrefix(ipnet)
	for ip := iprange.From(); ipnet.Contains(ip); ip = ip.Next() {
		ipstring := ip.String()
		_, ok := prefix.ips[ipstring]
		if ok {
			continue
		}
		return i.acquireAndStore(ctx, namespace, prefix, ip)
	}

	return nil, fmt.Errorf("%w: no more ips in prefix: %s left, length of prefix.ips: %d", ErrNoIPAvailable, prefix.Cidr, len(prefix.ips))
}

func (i *ipamer) acquireAndStore(ctx context.Context, namespace string, prefix *Prefix, ip netip.Addr) (*IP, error) {
	acquired := &IP{
		IP:           ip,
		ParentPrefix: prefix.Cidr,
	}
	prefix.ips[ip.String()] = true
	_, err := i.storage.UpdatePrefix(ctx, *prefix, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to persist acquired ip:%v error:%w", prefix, err)
	}
	return acquired, nil
}

func (i *ipamer) AcquireIP(ctx context.Context, prefixCidr string) (*IP, error) {
	return i.AcquireSpecificIP(ctx, prefixCidr, "")
}

func (i *ipamer) ReleaseIP(ctx context.Context, ip *IP) (*Prefix, error) {
	err := i.ReleaseIPFromPrefix(ctx, ip.ParentPrefix, ip.IP.String())
	if err != nil {
		return nil, err
	}
	return i.PrefixFrom(ctx, ip.ParentPrefix)
}

func (i *ipamer) ReleaseIPFromPrefix(ctx context.Context, prefixCidr, ip string) error {
	namespace := namespaceFromContext(ctx)
	return retryOnOptimisticLock(func() error {
		return i.releaseIPFromPrefixInternal(ctx, namespace, prefixCidr, ip)
	})
}

// releaseIPFromPrefixInternal will release the given IP for later usage.
func (i *ipamer) releaseIPFromPrefixInternal(ctx context.Context, namespace, prefixCidr, ip string) error {
	prefix, err := i.PrefixFrom(ctx, prefixCidr)
	if err != nil {
		return fmt.Errorf("%w: unable to find prefix for cidr:%s error:%s", ErrNotFound, prefixCidr, err.Error())
	}
	if prefix == nil {
		return fmt.Errorf("%w: unable to find prefix for cidr:%s", ErrNotFound, prefixCidr)
	}
	_, ok := prefix.ips[ip]
	if !ok {
		return fmt.Errorf("%w: unable to release ip:%s because it is not allocated in prefix:%s", ErrNotFound, ip, prefixCidr)
	}
	delete(prefix.ips, ip)
	_, err = i.storage.UpdatePrefix(ctx, *prefix, namespace)
	if err != nil {
		return fmt.Errorf("unable to release ip %v:%w", ip, err)
	}
	return nil
}

// PrefixesOverlapping will check if one ore more prefix of newPrefixes is overlapping
// with one of existingPrefixes
func PrefixesOverlapping(existingPrefixes []string, newPrefixes []string) error {
	for _, ep := range existingPrefixes {
		eip, err := netip.ParsePrefix(ep)
		if err != nil {
			return fmt.Errorf("parsing prefix %s failed:%w", ep, err)
		}
		for _, np := range newPrefixes {
			nip, err := netip.ParsePrefix(np)
			if err != nil {
				return fmt.Errorf("parsing prefix %s failed:%w", np, err)
			}
			if eip.Overlaps(nip) || nip.Overlaps(eip) {
				return fmt.Errorf("%s overlaps %s", nip, eip)
			}
		}
	}
	return nil
}

// newPrefix create a new Prefix from a string notation.
func (i *ipamer) newPrefix(cidr, parentCidr string) (*Prefix, error) {
	ipnet, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cidr:%s %w", cidr, err)
	}
	if parentCidr != "" {
		ipnetParent, err := netip.ParsePrefix(parentCidr)
		if err != nil {
			return nil, fmt.Errorf("unable to parse parent cidr:%s %w", cidr, err)
		}
		parentCidr = ipnetParent.Masked().String()
	}

	p := &Prefix{
		Cidr:                   ipnet.Masked().String(),
		ParentCidr:             parentCidr,
		ips:                    make(map[string]bool),
		availableChildPrefixes: make(map[string]bool),
		isParent:               false,
	}

	// FIXME: should this be done by the user ?
	// First ip in the prefix and broadcast is blocked.
	iprange := netipx.RangeOfPrefix(ipnet)
	p.ips[iprange.From().String()] = true
	if ipnet.Addr().Is4() {
		// broadcast is ipv4 only
		p.ips[iprange.To().String()] = true
	}

	return p, nil
}

func (i *ipamer) Dump(ctx context.Context) (string, error) {
	// FIXME must dump all namespaces
	return i.NamespacedDump(ctx, defaultNamespace)
}

func (i *ipamer) NamespacedDump(ctx context.Context, namespace string) (string, error) {
	pfxs, err := i.storage.ReadAllPrefixes(ctx, namespace)
	if err != nil {
		return "", err
	}
	js, err := pfxs.toJSON()
	if err != nil {
		return "", err
	}
	return string(js), nil
}

func (i *ipamer) Load(ctx context.Context, dump string) error {
	// FIXME must load all namespaces
	return i.NamespacedLoad(ctx, defaultNamespace, dump)
}

func (i *ipamer) NamespacedLoad(ctx context.Context, namespace, dump string) error {
	existingpfxs, err := i.storage.ReadAllPrefixes(ctx, namespace)
	if err != nil {
		return err
	}
	if len(existingpfxs) > 0 {
		return fmt.Errorf("prefixes exist, please drop existing data before loading")
	}
	pfxs, err := fromJSONs([]byte(dump))
	if err != nil {
		return err
	}
	err = i.storage.DeleteAllPrefixes(ctx, namespace)
	if err != nil {
		return err
	}
	for _, pfx := range pfxs {
		_, err = i.storage.CreatePrefix(ctx, pfx, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadAllPrefixCidrs retrieves all existing Prefix CIDRs from the underlying storage
func (i *ipamer) ReadAllPrefixCidrs(ctx context.Context) ([]string, error) {
	return i.ReadAllNamespacedPrefixCidrs(ctx, defaultNamespace)
}

// ReadAllNamespacedPrefixCidrs retrieves all existing Prefix CIDRs from the underlying storage
func (i *ipamer) ReadAllNamespacedPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	return i.storage.ReadAllPrefixCidrs(ctx, namespace)
}

// CreateNamespaces creates a namespace with the given name.
func (i *ipamer) CreateNamespace(ctx context.Context, namespace string) error {
	return i.storage.CreateNamespace(ctx, namespace)
}

// ListNamespaces returns a list of all namespaces.
func (i *ipamer) ListNamespaces(ctx context.Context) ([]string, error) {
	return i.storage.ListNamespaces(ctx)
}

// DeleteNamespace deletes a namespace.
func (i *ipamer) DeleteNamespace(ctx context.Context, namespace string) error {
	prefixes, err := i.storage.ReadAllPrefixes(ctx, namespace)
	if err != nil {
		return err
	}
	if len(prefixes) > 0 {
		return fmt.Errorf("cannot delete namespace with allocated prefixes")
	}
	return i.storage.DeleteNamespace(ctx, namespace)
}

func (p *Prefix) String() string {
	return p.Cidr
}

func (u *Usage) String() string {
	if u.AcquiredPrefixes == 0 {
		return fmt.Sprintf("ip:%d/%d", u.AcquiredIPs, u.AvailableIPs)
	}
	return fmt.Sprintf("ip:%d/%d prefixes alloc:%d avail:%d", u.AcquiredIPs, u.AvailableIPs, u.AcquiredPrefixes, u.AvailableSmallestPrefixes)
}

// Network return the net.IP part of the Prefix
func (p *Prefix) Network() (netip.Addr, error) {
	ipprefix, err := netip.ParsePrefix(p.Cidr)
	if err != nil {
		return netip.Addr{}, err
	}
	return ipprefix.Addr(), nil
}

// hasIPs will return true if there are allocated IPs
func (p *Prefix) hasIPs() bool {
	ipprefix, err := netip.ParsePrefix(p.Cidr)
	if err != nil {
		return false
	}
	if ipprefix.Addr().Is4() && len(p.ips) > 2 {
		return true
	}
	if ipprefix.Addr().Is6() && len(p.ips) > 1 {
		return true
	}
	return false
}

// availableips return the number of ips available in this Prefix
func (p *Prefix) availableips() uint64 {
	ipprefix, err := netip.ParsePrefix(p.Cidr)
	if err != nil {
		return 0
	}
	// We don't report more than 2^31 available IPs by design
	if (ipprefix.Addr().BitLen() - ipprefix.Bits()) > 31 {
		return math.MaxInt32
	}
	return 1 << (ipprefix.Addr().BitLen() - ipprefix.Bits())
}

// acquiredips return the number of ips acquired in this Prefix
func (p *Prefix) acquiredips() uint64 {
	return uint64(len(p.ips))
}

// availablePrefixes will return the amount of prefixes allocatable and the amount of smallest 2 bit prefixes
func (p *Prefix) availablePrefixes() (uint64, []string) {
	prefix, err := netip.ParsePrefix(p.Cidr)
	if err != nil {
		return 0, nil
	}
	var ipsetBuilder netipx.IPSetBuilder
	ipsetBuilder.AddPrefix(prefix)
	for cp, available := range p.availableChildPrefixes {
		if available {
			continue
		}
		ipprefix, err := netip.ParsePrefix(cp)
		if err != nil {
			continue
		}
		ipsetBuilder.RemovePrefix(ipprefix)
	}

	ipset, err := ipsetBuilder.IPSet()
	if err != nil {
		return 0, []string{}
	}

	// Only 2 Bit Prefixes are usable, set max bits available 2 less than max in family
	maxBits := prefix.Addr().BitLen() - 2
	pfxs := ipset.Prefixes()
	totalAvailable := uint64(0)
	availablePrefixes := []string{}
	for _, pfx := range pfxs {
		bits := maxBits - pfx.Bits()
		if bits < 0 {
			continue
		}
		// same as: totalAvailable += uint64(math.Pow(float64(2), float64(maxBits-pfx.Bits)))
		totalAvailable += 1 << bits
		availablePrefixes = append(availablePrefixes, pfx.String())
	}
	// we are not reporting more that 2^31 available prefixes
	if totalAvailable > math.MaxInt32 {
		totalAvailable = math.MaxInt32
	}
	return totalAvailable, availablePrefixes
}

// acquiredPrefixes return the amount of acquired prefixes of this prefix if this is a parent prefix
func (p *Prefix) acquiredPrefixes() uint64 {
	var count uint64
	for _, available := range p.availableChildPrefixes {
		if !available {
			count++
		}
	}
	return count
}

// Usage report Prefix usage.
func (p *Prefix) Usage() Usage {
	sp, ap := p.availablePrefixes()
	return Usage{
		AvailableIPs:              p.availableips(),
		AcquiredIPs:               p.acquiredips(),
		AcquiredPrefixes:          p.acquiredPrefixes(),
		AvailableSmallestPrefixes: sp,
		AvailablePrefixes:         ap,
	}
}

// retries the given function if the reported error is an OptimisticLockError
// with ten attempts and jitter delay ~100ms
// returns only error of last failed attempt
func retryOnOptimisticLock(retryableFunc retry.RetryableFunc) error {

	return retry.Do(
		retryableFunc,
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, ErrOptimisticLockError)
		}),
		retry.Attempts(10),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.LastErrorOnly(true))
}

func namespaceFromContext(ctx context.Context) string {
	raw := ctx.Value(namespaceContextKey{})
	if raw == nil {
		return defaultNamespace
	}
	if ns, ok := raw.(string); ok {
		return ns
	}
	return defaultNamespace
}
