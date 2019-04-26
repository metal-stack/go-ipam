package ipam

import (
	"fmt"
	"github.com/google/uuid"
)

var (
	networks = make(map[string]*Network)
	prefixes = make(map[string]*Prefix)
	ips      = make(map[string]*IP)
)

type memory struct{}

func (m memory) CreateNetwork(network *Network) (*Network, error) {
	if network.ID != "" {
		return nil, fmt.Errorf("network already created:%v", network)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	network.ID = id.String()
	networks[network.ID] = network
	return network, nil
}
func (m memory) ReadNetwork(id string) (*Network, error) {
	return networks[id], nil
}
func (m memory) ReadAllNetworks() ([]*Network, error) {
	nw := make([]*Network, 0, len(networks))
	for _, v := range networks {
		nw = append(nw, v)
	}
	return nw, nil
}
func (m memory) UpdateNetwork(network *Network) (*Network, error) {
	if network.ID == "" {
		return nil, fmt.Errorf("network not created:%v", network)
	}
	_, ok := networks[network.ID]
	if !ok {
		return nil, fmt.Errorf("network not found:%v", network)
	}
	networks[network.ID] = network
	return network, nil
}

func (m memory) DeleteNetwork(network *Network) (*Network, error) {
	delete(networks, network.ID)
	return network, nil
}

func (m memory) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	_, ok := prefixes[prefix.Cidr]
	if ok {
		return nil, fmt.Errorf("prefix already created:%v", prefix)
	}
	prefixes[prefix.Cidr] = prefix
	return prefix, nil
}
func (m memory) ReadPrefix(prefix string) (*Prefix, error) {
	return prefixes[prefix], nil
}
func (m memory) ReadAllPrefixes() ([]*Prefix, error) {
	ps := make([]*Prefix, 0, len(prefixes))
	for _, v := range prefixes {
		ps = append(ps, v)
	}
	return ps, nil
}
func (m memory) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	if prefix.Cidr == "" {
		return nil, fmt.Errorf("prefix not present:%v", prefix)
	}
	_, ok := prefixes[prefix.Cidr]
	if !ok {
		return nil, fmt.Errorf("prefix not found:%v", prefix)
	}
	prefixes[prefix.Cidr] = prefix
	return prefix, nil
}
func (m memory) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	delete(prefixes, prefix.Cidr)
	return prefix, nil
}
