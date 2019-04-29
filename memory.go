package ipam

import (
	"fmt"

	"github.com/google/uuid"
)

type memory struct {
	networks map[string]*Network
	prefixes map[string]*Prefix
}

func NewMemory() *memory {
	networks := make(map[string]*Network)
	prefixes := make(map[string]*Prefix)
	return &memory{
		networks: networks,
		prefixes: prefixes,
	}

}

func (m *memory) CreateNetwork(network *Network) (*Network, error) {
	if network.ID != "" {
		return nil, fmt.Errorf("network already created:%v", network)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	network.ID = id.String()
	m.networks[network.ID] = network
	return network, nil
}
func (m *memory) ReadNetwork(id string) (*Network, error) {
	return m.networks[id], nil
}
func (m *memory) ReadAllNetworks() ([]*Network, error) {
	nw := make([]*Network, 0, len(m.networks))
	for _, v := range m.networks {
		nw = append(nw, v)
	}
	return nw, nil
}
func (m *memory) UpdateNetwork(network *Network) (*Network, error) {
	if network.ID == "" {
		return nil, fmt.Errorf("network not created:%v", network)
	}
	_, ok := m.networks[network.ID]
	if !ok {
		return nil, fmt.Errorf("network not found:%v", network)
	}
	m.networks[network.ID] = network
	return network, nil
}

func (m *memory) DeleteNetwork(network *Network) (*Network, error) {
	delete(m.networks, network.ID)
	return network, nil
}

func (m *memory) CreatePrefix(prefix *Prefix) (*Prefix, error) {
	_, ok := m.prefixes[prefix.Cidr]
	if ok {
		return nil, fmt.Errorf("prefix already created:%v", prefix)
	}
	m.prefixes[prefix.Cidr] = prefix
	return prefix, nil
}
func (m *memory) ReadPrefix(prefix string) (*Prefix, error) {
	return m.prefixes[prefix], nil
}
func (m *memory) ReadAllPrefixes() ([]*Prefix, error) {
	ps := make([]*Prefix, 0, len(m.prefixes))
	for _, v := range m.prefixes {
		ps = append(ps, v)
	}
	return ps, nil
}
func (m *memory) UpdatePrefix(prefix *Prefix) (*Prefix, error) {
	if prefix.Cidr == "" {
		return nil, fmt.Errorf("prefix not present:%v", prefix)
	}
	_, ok := m.prefixes[prefix.Cidr]
	if !ok {
		return nil, fmt.Errorf("prefix not found:%v", prefix)
	}
	m.prefixes[prefix.Cidr] = prefix
	return prefix, nil
}
func (m *memory) DeletePrefix(prefix *Prefix) (*Prefix, error) {
	delete(m.prefixes, prefix.Cidr)
	return prefix, nil
}
