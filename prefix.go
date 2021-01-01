package ipam

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"inet.af/netaddr"
)

var (
	// ErrNotFound is returned if prefix or cidr was not found
	ErrNotFound NotFoundError
	// ErrNoIPAvailable is returned if no IP is available anymore
	ErrNoIPAvailable NoIPAvailableError
)

// Prefix is a expression of a ip with length and forms a classless network.
type Prefix struct {
	Cidr                   string          // The Cidr of this prefix
	ParentCidr             string          // if this prefix is a child this is a pointer back
	isParent               bool            // if this Prefix has child prefixes, this is set to true
	availableChildPrefixes map[string]bool // available child prefixes of this prefix
	ips                    map[string]bool // The ips contained in this prefix
	version                int64           // version is used for optimistic locking
}

// DeepCopy to a new Prefix
func (p Prefix) DeepCopy() *Prefix {
	return &Prefix{
		Cidr:                   p.Cidr,
		ParentCidr:             p.ParentCidr,
		isParent:               p.isParent,
		availableChildPrefixes: copyMap(p.availableChildPrefixes),
		ips:                    copyMap(p.ips),
		version:                p.version,
	}
}

// GobEncode implements GobEncode for Prefix
func (p *Prefix) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(p.availableChildPrefixes)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(p.isParent)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(p.ips)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(p.version)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(p.Cidr)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(p.ParentCidr)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// GobDecode implements GobDecode for Prefix
func (p *Prefix) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err := decoder.Decode(&p.availableChildPrefixes)
	if err != nil {
		return err
	}
	err = decoder.Decode(&p.isParent)
	if err != nil {
		return err
	}
	err = decoder.Decode(&p.ips)
	if err != nil {
		return err
	}
	err = decoder.Decode(&p.version)
	if err != nil {
		return err
	}
	err = decoder.Decode(&p.Cidr)
	if err != nil {
		return err
	}
	return decoder.Decode(&p.ParentCidr)
}

func copyMap(m map[string]bool) map[string]bool {
	cm := make(map[string]bool, len(m))
	for k, v := range m {
		cm[k] = v
	}
	return cm
}

// Usage of ips and child Prefixes of a Prefix
type Usage struct {
	AvailableIPs      uint64
	AcquiredIPs       uint64
	AvailablePrefixes []AvailablePrefix
	AcquiredPrefixes  uint64
}

func (a AvailablePrefix) String() string {
	return fmt.Sprintf("/%d:%d", a.PrefixLength, a.Count)
}

// AvailablePrefix count by length
type AvailablePrefix struct {
	PrefixLength uint8
	Count        uint32
}

func (i *ipamer) NewPrefix(cidr string) (*Prefix, error) {
	p, err := i.newPrefix(cidr)
	if err != nil {
		return nil, err
	}
	newPrefix, err := i.storage.CreatePrefix(*p)
	if err != nil {
		return nil, err
	}

	return &newPrefix, nil
}

func (i *ipamer) DeletePrefix(cidr string) (*Prefix, error) {
	p := i.PrefixFrom(cidr)
	if p == nil {
		return nil, fmt.Errorf("%w: delete prefix:%s", ErrNotFound, cidr)
	}
	if len(p.ips) > 2 {
		return nil, fmt.Errorf("prefix %s has ips, delete prefix not possible", p.Cidr)
	}
	prefix, err := i.storage.DeletePrefix(*p)
	if err != nil {
		return nil, fmt.Errorf("delete prefix:%s %v", cidr, err)
	}

	return &prefix, nil
}

func (i *ipamer) AcquireChildPrefix(parentCidr string, length uint8) (*Prefix, error) {
	var prefix *Prefix
	return prefix, retryOnOptimisticLock(func() error {
		var err error
		prefix, err = i.acquireChildPrefixInternal(parentCidr, length)
		return err
	})
}

// acquireChildPrefixInternal will return a Prefix with a smaller length from the given Prefix.
func (i *ipamer) acquireChildPrefixInternal(parentCidr string, length uint8) (*Prefix, error) {
	prefix := i.PrefixFrom(parentCidr)
	if prefix == nil {
		return nil, fmt.Errorf("unable to find prefix for cidr:%s", parentCidr)
	}
	if len(prefix.ips) > 2 {
		return nil, fmt.Errorf("prefix %s has ips, acquire child prefix not possible", prefix.Cidr)
	}
	ipprefix, err := netaddr.ParseIPPrefix(prefix.Cidr)
	if err != nil {
		return nil, err
	}
	if ipprefix.Bits >= length {
		return nil, fmt.Errorf("given length:%d must be greater than prefix length:%d", length, ipprefix.Bits)
	}

	var ipset netaddr.IPSet
	ipset.AddPrefix(ipprefix)
	for cp, available := range prefix.availableChildPrefixes {
		if available {
			continue
		}
		ipprefix, err := netaddr.ParseIPPrefix(cp)
		if err != nil {
			return nil, err
		}
		ipset.RemovePrefix(ipprefix)
	}

	cp, ok := ipset.RemoveFreePrefix(length)
	if !ok {
		return nil, fmt.Errorf("no prefix found in %s with length:%d", parentCidr, length)
	}

	child := &Prefix{
		Cidr:       cp.String(),
		ParentCidr: parentCidr,
	}

	prefix.availableChildPrefixes[child.Cidr] = false
	prefix.isParent = true

	_, err = i.storage.UpdatePrefix(*prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update parent prefix:%v", prefix)
	}
	child, err = i.NewPrefix(child.Cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to persist created child:%v", err)
	}
	child.ParentCidr = prefix.Cidr
	_, err = i.storage.UpdatePrefix(*child)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update parent prefix:%v", child)
	}

	return child, nil
}

func (i *ipamer) ReleaseChildPrefix(child *Prefix) error {
	return retryOnOptimisticLock(func() error {
		return i.releaseChildPrefixInternal(child)
	})
}

// releaseChildPrefixInternal will mark this child Prefix as available again.
func (i *ipamer) releaseChildPrefixInternal(child *Prefix) error {
	parent := i.PrefixFrom(child.ParentCidr)

	if parent == nil {
		return fmt.Errorf("prefix %s is no child prefix", child.Cidr)
	}
	if len(child.ips) > 2 {
		return fmt.Errorf("prefix %s has ips, deletion not possible", child.Cidr)
	}

	parent.availableChildPrefixes[child.Cidr] = true
	_, err := i.DeletePrefix(child.Cidr)
	if err != nil {
		return fmt.Errorf("unable to release prefix %v:%v", child, err)
	}
	_, err = i.storage.UpdatePrefix(*parent)
	if err != nil {
		return fmt.Errorf("unable to release prefix %v:%v", child, err)
	}
	return nil
}

func (i *ipamer) PrefixFrom(cidr string) *Prefix {
	prefix, err := i.storage.ReadPrefix(cidr)
	if err != nil {
		return nil
	}
	return &prefix
}

func (i *ipamer) AcquireSpecificIP(prefixCidr, specificIP string) (*IP, error) {
	var ip *IP
	return ip, retryOnOptimisticLock(func() error {
		var err error
		ip, err = i.acquireSpecificIPInternal(prefixCidr, specificIP)
		return err
	})
}

// acquireSpecificIPInternal will acquire given IP and mark this IP as used, if already in use, return nil.
// If specificIP is empty, the next free IP is returned.
// If there is no free IP an NoIPAvailableError is returned.
// If the Prefix is not found an NotFoundError is returned.
func (i *ipamer) acquireSpecificIPInternal(prefixCidr, specificIP string) (*IP, error) {
	prefix := i.PrefixFrom(prefixCidr)
	if prefix == nil {
		return nil, fmt.Errorf("%w: unable to find prefix for cidr:%s", ErrNotFound, prefixCidr)
	}
	if prefix.isParent {
		return nil, fmt.Errorf("prefix %s has childprefixes, acquire ip not possible", prefix.Cidr)
	}
	var acquired *IP
	ipnet, err := netaddr.ParseIPPrefix(prefix.Cidr)
	if err != nil {
		return nil, err
	}

	if specificIP != "" {
		specificIPnet, err := netaddr.ParseIP(specificIP)
		if err != nil {
			return nil, fmt.Errorf("given ip:%s in not valid", specificIP)
		}
		if !ipnet.Contains(specificIPnet) {
			return nil, fmt.Errorf("given ip:%s is not in %s", specificIP, prefixCidr)
		}
	}

	for ip := ipnet.Range().From; ipnet.Contains(ip); ip = ip.Next() {
		_, ok := prefix.ips[ip.String()]
		if ok {
			continue
		}
		if specificIP == "" || specificIP == ip.String() {
			acquired = &IP{
				IP:           ip,
				ParentPrefix: prefix.Cidr,
			}
			prefix.ips[ip.String()] = true
			_, err := i.storage.UpdatePrefix(*prefix)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to persist acquired ip:%v", prefix)
			}
			return acquired, nil
		}
	}

	return nil, fmt.Errorf("%w: no more ips in prefix: %s left, length of prefix.ips: %d", ErrNoIPAvailable, prefix.Cidr, len(prefix.ips))
}

func (i *ipamer) AcquireIP(prefixCidr string) (*IP, error) {
	return i.AcquireSpecificIP(prefixCidr, "")
}

func (i *ipamer) ReleaseIP(ip *IP) (*Prefix, error) {
	err := i.ReleaseIPFromPrefix(ip.ParentPrefix, ip.IP.String())
	prefix := i.PrefixFrom(ip.ParentPrefix)
	return prefix, err
}

func (i *ipamer) ReleaseIPFromPrefix(prefixCidr, ip string) error {
	return retryOnOptimisticLock(func() error {
		return i.releaseIPFromPrefixInternal(prefixCidr, ip)
	})
}

// releaseIPFromPrefixInternal will release the given IP for later usage.
func (i *ipamer) releaseIPFromPrefixInternal(prefixCidr, ip string) error {
	prefix := i.PrefixFrom(prefixCidr)
	if prefix == nil {
		return fmt.Errorf("%w: unable to find prefix for cidr:%s", ErrNotFound, prefixCidr)
	}
	_, ok := prefix.ips[ip]
	if !ok {
		return fmt.Errorf("%w: unable to release ip:%s because it is not allocated in prefix:%s", ErrNotFound, ip, prefixCidr)
	}
	delete(prefix.ips, ip)
	_, err := i.storage.UpdatePrefix(*prefix)
	if err != nil {
		return fmt.Errorf("unable to release ip %v:%w", ip, err)
	}
	return nil
}

func (i *ipamer) PrefixesOverlapping(existingPrefixes []string, newPrefixes []string) error {
	for _, ep := range existingPrefixes {
		eip, eipnet, err := net.ParseCIDR(ep)
		if err != nil {
			return fmt.Errorf("parsing prefix %s failed:%v", ep, err)
		}
		for _, np := range newPrefixes {
			nip, nipnet, err := net.ParseCIDR(np)
			if err != nil {
				return fmt.Errorf("parsing prefix %s failed:%v", np, err)
			}
			if eipnet.Contains(nip) || nipnet.Contains(eip) {
				return fmt.Errorf("%s overlaps %s", np, ep)
			}
		}
	}

	return nil
}

// newPrefix create a new Prefix from a string notation.
func (i *ipamer) newPrefix(cidr string) (*Prefix, error) {
	ipnet, err := netaddr.ParseIPPrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cidr:%s %v", cidr, err)
	}
	p := &Prefix{
		Cidr:                   cidr,
		ips:                    make(map[string]bool),
		availableChildPrefixes: make(map[string]bool),
	}

	// FIXME: should this be done by the user ?
	// First ip in the prefix and broadcast is blocked.
	p.ips[ipnet.Range().From.String()] = true
	if ipnet.IP.Is4() {
		// broadcast is ipv4 only
		p.ips[ipnet.Range().To.String()] = true
	}

	return p, nil
}

func (p *Prefix) String() string {
	return p.Cidr
}

func (u *Usage) String() string {
	if u.AcquiredPrefixes == 0 {
		return fmt.Sprintf("ip:%d/%d", u.AcquiredIPs, u.AvailableIPs)
	}
	avpfxs := []string{}
	for _, a := range u.AvailablePrefixes {
		avpfxs = append(avpfxs, a.String())
	}
	return fmt.Sprintf("ip:%d/%d prefixes alloc:%d avail:%s", u.AcquiredIPs, u.AvailableIPs, u.AcquiredPrefixes, strings.Join(avpfxs, ","))
}

// Network return the net.IP part of the Prefix
func (p *Prefix) Network() (net.IP, error) {
	ipprefix, err := netaddr.ParseIPPrefix(p.Cidr)
	if err != nil {
		return nil, err
	}
	return ipprefix.IPNet().IP, nil
}

// availableips return the number of ips available in this Prefix
func (p *Prefix) availableips() uint64 {
	ipprefix, err := netaddr.ParseIPPrefix(p.Cidr)
	if err != nil {
		return 0
	}
	var bits uint8
	if ipprefix.IP.Is4() {
		bits = 32
	}
	if ipprefix.IP.Is6() {
		bits = 128
	}
	return uint64(math.Pow(float64(2), float64(bits-ipprefix.Bits)))
}

// acquiredips return the number of ips acquired in this Prefix
func (p *Prefix) acquiredips() uint64 {
	return uint64(len(p.ips))
}

// availablePrefixes return the amount of possible prefixes of this prefix if this is a parent prefix
func (p *Prefix) availablePrefixes() []AvailablePrefix {
	// TODO much too complex implementation
	prefix, err := netaddr.ParseIPPrefix(p.Cidr)
	if err != nil {
		// TODO howto handle errors here
		return nil
	}
	var ipset netaddr.IPSet
	ipset.AddPrefix(prefix)
	for cp, available := range p.availableChildPrefixes {
		if available {
			continue
		}
		ipprefix, err := netaddr.ParseIPPrefix(cp)
		if err != nil {
			// TODO howto handle errors here
			return nil
		}
		ipset.RemovePrefix(ipprefix)
	}

	pfxs := ipset.Prefixes()
	apfxs := make(map[uint8]uint32)
	for _, pfx := range pfxs {
		count, ok := apfxs[pfx.Bits]
		if !ok {
			apfxs[pfx.Bits] = 1
			continue
		}
		apfxs[pfx.Bits] = count + 1
	}
	availablePrefixes := []AvailablePrefix{}
	for bits, count := range apfxs {
		availablePrefix := AvailablePrefix{
			PrefixLength: bits,
			Count:        count,
		}
		availablePrefixes = append(availablePrefixes, availablePrefix)
	}
	sort.Slice(availablePrefixes, func(i, j int) bool {
		return availablePrefixes[i].PrefixLength > availablePrefixes[j].PrefixLength
	})
	return availablePrefixes
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
	return Usage{
		AvailableIPs:      p.availableips(),
		AcquiredIPs:       p.acquiredips(),
		AvailablePrefixes: p.availablePrefixes(),
		AcquiredPrefixes:  p.acquiredPrefixes(),
	}
}

// NoIPAvailableError indicates that the acquire-operation could not be executed
// because the specified prefix has no free IP anymore.
type NoIPAvailableError struct {
}

func (o NoIPAvailableError) Error() string {
	return "NoIPAvailableError"
}

// NotFoundError is raised if the given Prefix or Cidr was not found
type NotFoundError struct {
}

func (o NotFoundError) Error() string {
	return "NotFound"
}

// retries the given function if the reported error is an OptimisticLockError
// with ten attempts and jitter delay ~100ms
// returns only error of last failed attempt
func retryOnOptimisticLock(retryableFunc retry.RetryableFunc) error {

	return retry.Do(
		retryableFunc,
		retry.RetryIf(func(err error) bool {
			_, isOptimisticLock := errors.Cause(err).(OptimisticLockError)
			return isOptimisticLock
		}),
		retry.Attempts(10),
		retry.DelayType(jitterDelay),
		retry.LastErrorOnly(true))
}

// jitter will add jitter to a time.Duration.
func jitter(d time.Duration) time.Duration {
	const jitter = 0.50
	jit := 1 + jitter*(rand.Float64()*2-1)
	return time.Duration(jit * float64(d))
}

// jitterDelay is a DelayType which varies delay in each iterations
func jitterDelay(_ uint, err error, config *retry.Config) time.Duration {
	// fields in config are private, so we hardcode the average delay duration
	return jitter(100 * time.Millisecond)
}
