package ipam

import (
	"fmt"
)

type memory struct {
	prefixes map[string]*Prefix
}

// NewMemory create a memory storage for ipam
func NewMemory() *memory {
	prefixes := make(map[string]*Prefix)
	return &memory{
		prefixes: prefixes,
	}
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
