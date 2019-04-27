package ipam

import (
	"fmt"
	"math"
	"net"
	"sync"
)

// Prefix is a expression of a ip with length and forms a classless network.
type Prefix struct {
	sync.Mutex
	Cidr                   string          // The Cidr of this prefix
	availableChildPrefixes map[string]bool // available child prefixes of this prefix
	childPrefixLength      int             // the length of the child prefixes
	ips                    map[string]IP   // The ips contained in this prefix
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

// AcquireChildPrefix will return a Prefix with a smaller length from the given Prefix.
// FIXME allow variable child prefix length
func (i *Ipamer) AcquireChildPrefix(prefix *Prefix, length int) (*Prefix, error) {
	prefix.Lock()
	defer prefix.Unlock()
	ipnet, err := prefix.IPNet()
	if err != nil {
		return nil, err
	}
	ones, size := ipnet.Mask.Size()
	if ones >= length {
		return nil, fmt.Errorf("given length:%d is smaller or equal of prefix length:%d", length, ones)
	}

	// If this is the first call, create a pool of available child prefixes with given length upfront
	if prefix.childPrefixLength == 0 {
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
			prefix.availableChildPrefixes[child.Cidr] = true

		}
		prefix.childPrefixLength = length
	}
	if prefix.childPrefixLength != length {
		return nil, fmt.Errorf("given length:%d is not equal to existing child prefix length:%d", length, prefix.childPrefixLength)
	}

	var child *Prefix
	for c, available := range prefix.availableChildPrefixes {
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

	prefix.availableChildPrefixes[child.Cidr] = false

	i.storage.UpdatePrefix(prefix)
	child, err = i.NewPrefix(child.Cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to persist created child:%v", err)
	}
	return child, nil
}

// ReleaseChildPrefix will mark this child Prefix as available again.
func (i *Ipamer) ReleaseChildPrefix(child *Prefix) error {
	parent := i.getParentPrefix(child)

	if parent == nil {
		return fmt.Errorf("given prefix is no child prefix")
	}
	parent.Lock()
	defer parent.Unlock()

	parent.availableChildPrefixes[child.Cidr] = true
	_, err := i.storage.UpdatePrefix(parent)
	if err != nil {
		return fmt.Errorf("unable to release prefix %v:%v", child, err)
	}
	return nil
}

// PrefixFrom will return a known Prefix
func (i *Ipamer) PrefixFrom(cidr string) *Prefix {
	prefixes, err := i.storage.ReadAllPrefixes()
	if err != nil {
		return nil
	}
	_, targetIpnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	for _, p := range prefixes {
		ipnet, err := p.IPNet()
		if err != nil {
			return nil
		}
		if ipnet.IP.String() == targetIpnet.IP.String() && ipnet.Mask.String() == targetIpnet.Mask.String() {
			return p
		}
	}
	return nil
}

// AcquireIP will return the next unused IP from this Prefix.
func (i *Ipamer) AcquireIP(prefix Prefix) (*IP, error) {
	prefix.Lock()
	defer prefix.Unlock()
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
		_, ok := prefix.ips[ip.String()]
		if !ok {
			acquired = &IP{
				IP:    ip,
				IPNet: ipnet,
			}
			prefix.ips[ip.String()] = *acquired
			_, err := i.storage.UpdatePrefix(&prefix)
			if err != nil {
				return nil, fmt.Errorf("unable to persist aquired ip:%v", err)
			}
			return acquired, nil
		}
	}
	return nil, nil
}

// ReleaseIP will release the given IP for later usage.
func (i *Ipamer) ReleaseIP(ip IP) error {
	prefix := i.getPrefixOfIP(&ip)
	return i.ReleaseIPFromPrefix(prefix, ip.IP.String())
}

// ReleaseIPFromPrefix will release the given IP for later usage.
func (i *Ipamer) ReleaseIPFromPrefix(prefix *Prefix, ip string) error {
	if prefix == nil {
		return fmt.Errorf("prefix is nil")
	}
	prefix.Lock()
	defer prefix.Unlock()

	_, ok := prefix.ips[ip]
	if !ok {
		return fmt.Errorf("unable to release ip:%s because it is not allocated in prefix:%s", ip, prefix.Cidr)
	}
	delete(prefix.ips, ip)
	_, err := i.storage.UpdatePrefix(prefix)
	if err != nil {
		return fmt.Errorf("unable to release ip %v:%v", ip, err)
	}
	return nil
}

func (i *Ipamer) getPrefixOfIP(ip *IP) *Prefix {
	prefixes, err := i.storage.ReadAllPrefixes()
	if err != nil {
		return nil
	}
	for _, p := range prefixes {
		ipnet, err := p.IPNet()
		if err != nil {
			return nil
		}
		if ipnet.Contains(ip.IP) && ipnet.Mask.String() == ip.IPNet.Mask.String() {
			return p
		}
	}
	return nil
}

func (i *Ipamer) getParentPrefix(prefix *Prefix) *Prefix {
	prefixes, err := i.storage.ReadAllPrefixes()
	if err != nil {
		return nil
	}
	targetIpnet, err := prefix.IPNet()
	if err != nil {
		return nil
	}
	for _, p := range prefixes {
		ipnet, err := p.IPNet()
		if err != nil {
			return nil
		}
		if ipnet.Contains(targetIpnet.IP) {
			return p
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
		ips:                    make(map[string]IP),
		availableChildPrefixes: make(map[string]bool),
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
	p.ips[network.String()] = IP{IP: network}
	p.ips[broadcast.IP.String()] = *broadcast

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
func (p *Prefix) AvailableIPs() int64 {
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
	count := int64(math.Pow(float64(2), float64(bits-ones)))
	return count
}

// AcquiredIPs return the number of IPs acquired in this Prefix
func (p *Prefix) AcquiredIPs() int {
	return len(p.ips)
}
