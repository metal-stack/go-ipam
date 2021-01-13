package ipam

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

type memory struct {
	prefixes map[string]Prefix
	lock     sync.RWMutex
}

// NewMemory create a memory storage for ipam
func NewMemory() Storage {
	prefixes := make(map[string]Prefix)
	return &memory{
		prefixes: prefixes,
		lock:     sync.RWMutex{},
	}
}

func (m *memory) CreatePrefix(prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.prefixes[prefix.Cidr]
	if ok {
		return Prefix{}, fmt.Errorf("prefix already created:%v", prefix)
	}
	m.prefixes[prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) ReadPrefix(prefix string) (Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	result, ok := m.prefixes[prefix]
	if !ok {
		return Prefix{}, errors.Errorf("Prefix %s not found", prefix)
	}
	return *result.deepCopy(), nil
}
func (m *memory) ReadAllPrefixes() ([]Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ps := make([]Prefix, 0, len(m.prefixes))
	for _, v := range m.prefixes {
		ps = append(ps, *v.deepCopy())
	}
	return ps, nil
}
func (m *memory) UpdatePrefix(prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if prefix.Cidr == "" {
		return Prefix{}, fmt.Errorf("prefix not present:%v", prefix)
	}
	_, ok := m.prefixes[prefix.Cidr]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}
	m.prefixes[prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) DeletePrefix(prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.prefixes, prefix.Cidr)
	return *prefix.deepCopy(), nil
}
