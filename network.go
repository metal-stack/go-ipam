package ipam

import "fmt"

// NewNetwork creates and persists a Network with the given Prefixes.
func (i *Ipamer) NewNetwork(prefixes ...Prefix) (*Network, error) {
	n, p, err := i.prefixesOverlapping(prefixes...)
	if err != nil {
		return nil, fmt.Errorf("unable to check for prefix overlap:%v", err)
	}
	if n != nil || p != nil {
		return nil, fmt.Errorf("prefix %s in network %s overlap", p.Cidr, n.ID)
	}
	network := &Network{
		Prefixes: prefixes,
	}
	nw, err := i.storage.CreateNetwork(network)
	return nw, err
}

func (i *Ipamer) prefixesOverlapping(prefixes ...Prefix) (*Network, *Prefix, error) {
	networks, err := i.Networks()
	if err != nil {
		return nil, nil, err
	}
	for _, network := range networks {
		for _, p := range network.Prefixes {
			pinet, err := p.IPNet()
			if err != nil {
				return nil, nil, err
			}
			for _, prefix := range prefixes {
				prefixinet, err := prefix.IPNet()
				if err != nil {
					return nil, nil, err
				}
				if pinet.Contains(prefixinet.IP) || prefixinet.Contains(pinet.IP) {
					return network, &prefix, nil
				}
			}
		}
	}
	return nil, nil, nil
}

// AddPrefix add a Prefix to a Network, Prefix must be stored before.
func (i *Ipamer) AddPrefix(network *Network, prefix *Prefix) (*Network, error) {
	n, p, err := i.prefixesOverlapping(*prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to check for prefix overlap:%v", err)
	}
	if n != nil || p != nil {
		return nil, fmt.Errorf("prefix %s in network %s overlap", p.Cidr, n.ID)
	}
	nw, err := i.storage.ReadNetwork(network.ID)
	if err != nil {
		return nil, err
	}
	p, err = i.storage.ReadPrefix(prefix.Cidr)
	if p != nil {
		return nil, fmt.Errorf("prefix: %v is not stored", prefix)
	}
	nw.Prefixes = append(nw.Prefixes, *p)
	newNetwork, err := i.storage.UpdateNetwork(nw)
	if err != nil {
		return nil, err
	}
	return newNetwork, nil
}

// DeleteNetwork will delete a Network
func (i *Ipamer) DeleteNetwork(network *Network) (*Network, error) {
	n, err := i.storage.DeleteNetwork(network)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// NetworkFrom returns a network for a id
func (i *Ipamer) NetworkFrom(id string) (*Network, error) {
	return i.storage.ReadNetwork(id)
}

// Networks returns a collection of all known Networks
func (i *Ipamer) Networks() ([]*Network, error) {
	return i.storage.ReadAllNetworks()
}
