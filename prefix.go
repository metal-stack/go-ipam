package ipam

import (
	"fmt"
	"net"
	"sync"
)

// Prefix is a expression of a ip with length and forms a classless network.
type Prefix struct {
	sync.Mutex
	Cidr                   string            // The Cidr of this prefix
	IPNet                  *net.IPNet        // the parsed IPNet of this prefix
	Network                net.IP            // IP of the network
	AvailableChildPrefixes map[string]Prefix // available child prefixes of this prefix
	AcquiredChildPrefixes  map[string]Prefix // acquired child prefixes of this prefix
	ChildPrefixLength      int               // the length of the child prefixes
	IPs                    map[string]IP     // The ips contained in this prefix
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
	ones, size := prefix.IPNet.Mask.Size()
	if ones >= length {
		return nil, fmt.Errorf("given length:%d is smaller or equal of prefix length:%d", length, ones)
	}

	// If this is the first call, create a pool of available child prefixes with given length upfront
	if prefix.ChildPrefixLength == 0 {
		// power of 2 :-(
		ip := prefix.IPNet.IP
		subnetCount := 1 << (uint(length - ones))
		for s := 0; s < subnetCount; s++ {
			newIP := &net.IPNet{
				IP:   insertNumIntoIP(ip, s, length),
				Mask: net.CIDRMask(length, size),
			}
			newCidr := newIP.String()
			child, err := i.newPrefix(newCidr)
			if err != nil {
				return nil, err
			}
			prefix.AvailableChildPrefixes[child.Cidr] = *child

		}
		prefix.ChildPrefixLength = length
	}
	if prefix.ChildPrefixLength != length {
		return nil, fmt.Errorf("given length:%d is not equal to existing child prefix length:%d", length, prefix.ChildPrefixLength)
	}

	var child *Prefix
	for _, v := range prefix.AvailableChildPrefixes {
		child = &v
		break
	}
	if child == nil {
		return nil, fmt.Errorf("no more child prefixes contained in prefix pool")
	}

	delete(prefix.AvailableChildPrefixes, child.Cidr)
	prefix.AcquiredChildPrefixes[child.Cidr] = *child

	i.storage.UpdatePrefix(prefix)
	child, err := i.NewPrefix(child.Cidr)
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

	delete(parent.AcquiredChildPrefixes, child.Cidr)
	parent.AvailableChildPrefixes[child.Cidr] = *child
	_, err := i.storage.UpdatePrefix(parent)
	if err != nil {
		parent.AcquiredChildPrefixes[child.Cidr] = *child
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
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	for _, p := range prefixes {
		if p.IPNet.IP.String() == ipnet.IP.String() && p.IPNet.Mask.String() == ipnet.Mask.String() {
			return p
		}
	}
	return nil
}

func (i *Ipamer) getPrefixOfIP(ip *IP) *Prefix {
	prefixes, err := i.storage.ReadAllPrefixes()
	if err != nil {
		return nil
	}
	for _, p := range prefixes {
		if p.IPNet.Contains(ip.IP) && p.IPNet.Mask.String() == ip.IPNet.Mask.String() {
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
	for _, p := range prefixes {
		if p.IPNet.Contains(prefix.IPNet.IP) {
			return p
		}
	}
	return nil
}

// NewPrefix create a new Prefix from a string notation.
func (i *Ipamer) newPrefix(cidr string) (*Prefix, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cidr:%s %v", cidr, err)
	}
	p := &Prefix{
		Cidr:                   cidr,
		IPNet:                  ipnet,
		Network:                ip.Mask(ipnet.Mask),
		IPs:                    make(map[string]IP),
		AvailableChildPrefixes: make(map[string]Prefix),
		AcquiredChildPrefixes:  make(map[string]Prefix),
	}

	broadcast := p.broadcast()
	// First IP in the prefix and Broadcast is blocked.
	p.IPs[p.Network.String()] = IP{IP: p.Network}
	p.IPs[broadcast.IP.String()] = broadcast

	return p, nil
}
