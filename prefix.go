package ipam

import (
	"fmt"
	"math"
	"net"
	"sync"
)

// Prefix is a expression of a ip with length and forms a classless network.
type Prefix struct {
	sync.Mutex             `json:"-"`
	Cidr                   string          // The Cidr of this prefix
	ParentCidr             string          // if this prefix is a child a pointer back
	AvailableChildPrefixes map[string]bool // available child prefixes of this prefix
	ChildPrefixLength      int             // the length of the child prefixes
	IPs                    map[string]IP   // The ips contained in this prefix
}

// Usage of IPs and child Prefixes of a Prefix
type Usage struct {
	AvailableIPs      uint64
	AcquiredIPs       uint64
	AvailablePrefixes uint64
	AcquiredPrefixes  uint64
}

// NewPrefix create a new Prefix from a string notation.
func (i *Ipamer) NewPrefix(cidr string) (*Prefix, error) {
	p, err := i.newPrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("NewPrefix:%s %v", cidr, err)
	}
	newPrefix, err := i.storage.CreatePrefix(p)
	if err != nil {
		return nil, fmt.Errorf("newPrefix:%s %v", cidr, err)
	}

	return newPrefix, nil
}

// DeletePrefix delete a Prefix from a string notation.
func (i *Ipamer) DeletePrefix(cidr string) (*Prefix, error) {
	p := i.PrefixFrom(cidr)
	if p == nil {
		return nil, fmt.Errorf("delete prefix:%s not found", cidr)
	}
	if len(p.IPs) > 2 {
		return nil, fmt.Errorf("prefix %s has ips, delete prefix not possible", p.Cidr)
	}
	prefix, err := i.storage.DeletePrefix(p)
	if err != nil {
		return nil, fmt.Errorf("delete prefix:%s %v", cidr, err)
	}

	return prefix, nil
}

// AcquireChildPrefix will return a Prefix with a smaller length from the given Prefix.
// FIXME allow variable child prefix length
func (i *Ipamer) AcquireChildPrefix(prefix *Prefix, length int) (*Prefix, error) {
	prefix.Lock()
	defer prefix.Unlock()
	if len(prefix.IPs) > 2 {
		return nil, fmt.Errorf("prefix %s has ips, acquire child prefix not possible", prefix.Cidr)
	}
	ipnet, err := prefix.IPNet()
	if err != nil {
		return nil, err
	}
	ones, size := ipnet.Mask.Size()
	if ones >= length {
		return nil, fmt.Errorf("given length:%d is smaller or equal of prefix length:%d", length, ones)
	}

	// If this is the first call, create a pool of available child prefixes with given length upfront
	if prefix.ChildPrefixLength == 0 {
		ip := ipnet.IP
		// FIXME use big.Int
		// power of 2 :-(
		// subnetCount := 1 << (uint(length - ones))
		subnetCount := int(math.Pow(float64(2), float64(length-ones)))
		for s := 0; s < subnetCount; s++ {
			ipPart, err := insertNumIntoIP(ip, s, length)
			if err != nil {
				return nil, err
			}
			newIP := &net.IPNet{
				IP:   *ipPart,
				Mask: net.CIDRMask(length, size),
			}
			newCidr := newIP.String()
			child, err := i.newPrefix(newCidr)
			if err != nil {
				return nil, err
			}
			prefix.AvailableChildPrefixes[child.Cidr] = true

		}
		prefix.ChildPrefixLength = length
	}
	if prefix.ChildPrefixLength != length {
		return nil, fmt.Errorf("given length:%d is not equal to existing child prefix length:%d", length, prefix.ChildPrefixLength)
	}

	var child *Prefix
	for c, available := range prefix.AvailableChildPrefixes {
		if !available {
			continue
		}
		child, err = i.newPrefix(c)
		if err != nil {
			continue
		}
		break
	}
	if child == nil {
		return nil, fmt.Errorf("no more child prefixes contained in prefix pool")
	}

	prefix.AvailableChildPrefixes[child.Cidr] = false

	i.storage.UpdatePrefix(prefix)
	child, err = i.NewPrefix(child.Cidr)
	child.ParentCidr = prefix.Cidr
	if err != nil {
		return nil, fmt.Errorf("unable to persist created child:%v", err)
	}
	return child, nil
}

// ReleaseChildPrefix will mark this child Prefix as available again.
func (i *Ipamer) ReleaseChildPrefix(child *Prefix) error {
	parent := i.PrefixFrom(child.ParentCidr)

	if parent == nil {
		return fmt.Errorf("given prefix is no child prefix")
	}
	if len(child.IPs) > 2 {
		return fmt.Errorf("prefix %s has ips, deletion not possible", child.Cidr)
	}

	parent.Lock()
	defer parent.Unlock()

	parent.AvailableChildPrefixes[child.Cidr] = true
	_, err := i.DeletePrefix(child.Cidr)
	if err != nil {
		return fmt.Errorf("unable to release prefix %v:%v", child, err)
	}
	_, err = i.storage.UpdatePrefix(parent)
	if err != nil {
		return fmt.Errorf("unable to release prefix %v:%v", child, err)
	}
	return nil
}

// PrefixFrom will return a known Prefix
func (i *Ipamer) PrefixFrom(cidr string) *Prefix {
	prefix, err := i.storage.ReadPrefix(cidr)
	if err != nil {
		return nil
	}
	return prefix
}

// AcquireIP will return the next unused IP from this Prefix.
func (i *Ipamer) AcquireIP(prefix *Prefix) (*IP, error) {
	prefix.Lock()
	defer prefix.Unlock()
	if prefix.ChildPrefixLength > 0 {
		return nil, fmt.Errorf("prefix %s has childprefixes, acquire ip not possible", prefix.Cidr)
	}
	var acquired *IP
	ipnet, err := prefix.IPNet()
	if err != nil {
		return nil, err
	}
	network, err := prefix.Network()
	if err != nil {
		return nil, err
	}
	for ip := network.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		_, ok := prefix.IPs[ip.String()]
		if !ok {
			acquired = &IP{
				IP:           ip,
				ParentPrefix: prefix.Cidr,
			}
			prefix.IPs[ip.String()] = *acquired
			_, err := i.storage.UpdatePrefix(prefix)
			if err != nil {
				return nil, fmt.Errorf("unable to persist aquired ip:%v", err)
			}
			return acquired, nil
		}
	}
	return nil, nil
}

// ReleaseIP will release the given IP for later usage.
func (i *Ipamer) ReleaseIP(ip *IP) error {
	prefix := i.PrefixFrom(ip.ParentPrefix)
	return i.ReleaseIPFromPrefix(prefix, ip.IP.String())
}

// ReleaseIPFromPrefix will release the given IP for later usage.
func (i *Ipamer) ReleaseIPFromPrefix(prefix *Prefix, ip string) error {
	if prefix == nil {
		return fmt.Errorf("prefix is nil")
	}
	prefix.Lock()
	defer prefix.Unlock()

	_, ok := prefix.IPs[ip]
	if !ok {
		return fmt.Errorf("unable to release ip:%s because it is not allocated in prefix:%s", ip, prefix.Cidr)
	}
	delete(prefix.IPs, ip)
	_, err := i.storage.UpdatePrefix(prefix)
	if err != nil {
		return fmt.Errorf("unable to release ip %v:%v", ip, err)
	}
	return nil
}

// PrefixesOverlapping will check if one ore more prefix of newPrefixes is overlapping
// with one of existingPrefixes
// FIXME should we change signature to PrefixOverlapping(newPrefix string) only
// and find all non superPrefixes ourselves
// that requires that newPrefix was not persisted before and we must implement .IPNet here as well.
func (i *Ipamer) PrefixesOverlapping(exitingPrefixes []string, newPrefixes []string) error {
	for _, p := range exitingPrefixes {
		existingPrefix := i.PrefixFrom(p)
		if existingPrefix == nil {
			return fmt.Errorf("unknown prefix %s", p)
		}
		existingPrefixNet, err := existingPrefix.IPNet()
		if err != nil {
			return err
		}
		for _, prefix := range newPrefixes {
			newPrefix := i.PrefixFrom(prefix)
			if existingPrefix == nil {
				return fmt.Errorf("unknown prefix %s", prefix)
			}
			newPrefixNet, err := newPrefix.IPNet()
			if err != nil {
				return err
			}
			if existingPrefixNet.Contains(newPrefixNet.IP) || newPrefixNet.Contains(existingPrefixNet.IP) {
				return fmt.Errorf("%s overlaps %s", newPrefix.Cidr, existingPrefix.Cidr)
			}
		}
	}

	return nil
}

// NewPrefix create a new Prefix from a string notation.
func (i *Ipamer) newPrefix(cidr string) (*Prefix, error) {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cidr:%s %v", cidr, err)
	}
	p := &Prefix{
		Cidr:                   cidr,
		IPs:                    make(map[string]IP),
		AvailableChildPrefixes: make(map[string]bool),
	}

	broadcast, err := p.broadcast()
	if err != nil {
		return nil, err
	}
	// First IP in the prefix and Broadcast is blocked.
	network, err := p.Network()
	if err != nil {
		return nil, err
	}
	p.IPs[network.String()] = IP{IP: network}
	p.IPs[broadcast.IP.String()] = *broadcast

	return p, nil
}

func (p *Prefix) broadcast() (*IP, error) {
	ipnet, err := p.IPNet()
	if err != nil {
		return nil, err
	}
	network, err := p.Network()
	if err != nil {
		return nil, err
	}
	mask := ipnet.Mask
	n := IP{IP: network}
	m := IP{IP: net.IP(mask)}

	broadcast := n.or(m.not())
	return &broadcast, nil
}

func (p *Prefix) String() string {
	return fmt.Sprintf("%s", p.Cidr)
}

func (u *Usage) String() string {
	if u.AvailablePrefixes == uint64(0) {
		return fmt.Sprintf("ip:%d/%d", u.AvailableIPs, u.AcquiredIPs)
	}
	return fmt.Sprintf("ip:%d/%d prefix:%d/%d", u.AvailableIPs, u.AcquiredIPs, u.AvailablePrefixes, u.AcquiredPrefixes)
}

// IPNet return the net.IPNet part of the Prefix
func (p *Prefix) IPNet() (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(p.Cidr)
	return ipnet, err
}

// Network return the net.IP part of the Prefix
func (p *Prefix) Network() (net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(p.Cidr)
	if err != nil {
		return nil, err
	}
	return ip.Mask(ipnet.Mask), nil
}

// AvailableIPs return the number of IPs available in this Prefix
func (p *Prefix) availableIPs() uint64 {
	_, ipnet, err := net.ParseCIDR(p.Cidr)
	if err != nil {
		return 0
	}
	var bits int
	if len(ipnet.IP) == net.IPv4len {
		bits = 32
	} else if len(ipnet.IP) == net.IPv6len {
		bits = 128
	}

	ones, _ := ipnet.Mask.Size()
	// FIXME use big.Int
	count := uint64(math.Pow(float64(2), float64(bits-ones)))
	return count
}

// AcquiredIPs return the number of IPs acquired in this Prefix
func (p *Prefix) acquiredIPs() uint64 {
	return uint64(len(p.IPs))
}

// AvailablePrefixes return the amount of possible prefixes of this prefix if this is a parent prefix
func (p *Prefix) availablePrefixes() uint64 {
	return uint64(len(p.AvailableChildPrefixes))
}

// AcquiredPrefixes return the amount of acquired prefixes of this prefix if this is a parent prefix
func (p *Prefix) acquiredPrefixes() uint64 {
	var count uint64
	for _, available := range p.AvailableChildPrefixes {
		if !available {
			count++
		}
	}
	return count
}

// Usage report Prefix usage.
func (p *Prefix) Usage() Usage {
	return Usage{
		AvailableIPs:      p.availableIPs(),
		AcquiredIPs:       p.acquiredIPs(),
		AvailablePrefixes: p.availablePrefixes(),
		AcquiredPrefixes:  p.acquiredPrefixes(),
	}

}
