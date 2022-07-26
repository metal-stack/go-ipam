package ipam

import (
	"context"
	"fmt"
	"sync"
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
func (m *memory) Name() string {
	return "memory"
}
func (m *memory) CreatePrefix(_ context.Context, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.prefixes[prefix.Cidr]
	if ok {
		return Prefix{}, fmt.Errorf("prefix already created:%v", prefix)
	}
	m.prefixes[prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) ReadPrefix(_ context.Context, prefix string) (Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	result, ok := m.prefixes[prefix]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix %s not found", prefix)
	}
	return *result.deepCopy(), nil
}
func (m *memory) DeleteAllPrefixes(_ context.Context) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	m.prefixes = make(map[string]Prefix)
	return nil
}
func (m *memory) ReadAllPrefixes(_ context.Context) (Prefixes, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ps := make(Prefixes, 0, len(m.prefixes))
	for _, v := range m.prefixes {
		ps = append(ps, *v.deepCopy())
	}
	return ps, nil
}
func (m *memory) ReadAllPrefixCidrs(_ context.Context) ([]string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ps := make([]string, 0, len(m.prefixes))
	for cidr := range m.prefixes {
		ps = append(ps, cidr)
	}
	return ps, nil
}
func (m *memory) UpdatePrefix(_ context.Context, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	oldVersion := prefix.version
	prefix.version = oldVersion + 1

	if prefix.Cidr == "" {
		return Prefix{}, fmt.Errorf("prefix not present:%v", prefix)
	}
	oldPrefix, ok := m.prefixes[prefix.Cidr]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}
	if oldPrefix.version != oldVersion {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}
	m.prefixes[prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) DeletePrefix(_ context.Context, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.prefixes, prefix.Cidr)
	return *prefix.deepCopy(), nil
}
