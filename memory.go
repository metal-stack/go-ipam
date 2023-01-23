package ipam

import (
	"context"
	"fmt"
	"sync"
)

type memory struct {
	prefixes map[string]map[string]Prefix
	lock     sync.RWMutex
}

// NewMemory create a memory storage for ipam
func NewMemory() Storage {
	prefixes := make(map[string]map[string]Prefix)
	return &memory{
		prefixes: prefixes,
		lock:     sync.RWMutex{},
	}
}
func (m *memory) Name() string {
	return "memory"
}
func (m *memory) CreatePrefix(_ context.Context, prefix Prefix, namespace string) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.prefixes[namespace]; !ok {
		m.prefixes[namespace] = make(map[string]Prefix)
	}
	_, ok := m.prefixes[namespace][prefix.Cidr]
	if ok {
		return Prefix{}, fmt.Errorf("prefix already created:%v", prefix)
	}
	m.prefixes[namespace][prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) ReadPrefix(_ context.Context, prefix, namespace string) (Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if _, ok := m.prefixes[namespace]; !ok {
		return Prefix{}, fmt.Errorf("prefix %s not found", prefix)
	}
	result, ok := m.prefixes[namespace][prefix]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix %s not found", prefix)
	}
	return *result.deepCopy(), nil
}

func (m *memory) DeleteAllPrefixes(_ context.Context, namespace string) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	m.prefixes[namespace] = make(map[string]Prefix)
	return nil
}

func (m *memory) ReadAllPrefixes(_ context.Context, namespace string) (Prefixes, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ps := make([]Prefix, 0, len(m.prefixes))
	if _, ok := m.prefixes[namespace]; !ok {
		return ps, nil
	}
	for _, v := range m.prefixes[namespace] {
		ps = append(ps, *v.deepCopy())
	}
	return ps, nil
}

func (m *memory) ReadAllPrefixCidrs(_ context.Context, namespace string) ([]string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if _, ok := m.prefixes[namespace]; !ok {
		return []string{}, nil
	}

	ps := make([]string, 0, len(m.prefixes[namespace]))
	for _, v := range m.prefixes[namespace] {
		ps = append(ps, v.Cidr)
	}
	return ps, nil
}

func (m *memory) UpdatePrefix(_ context.Context, prefix Prefix, namespace string) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	oldVersion := prefix.version
	prefix.version = oldVersion + 1

	if prefix.Cidr == "" {
		return Prefix{}, fmt.Errorf("prefix not present:%v", prefix)
	}

	if _, ok := m.prefixes[namespace]; !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}
	oldPrefix, ok := m.prefixes[namespace][prefix.Cidr]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}

	if oldPrefix.version != oldVersion {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}
	m.prefixes[namespace][prefix.Cidr] = *prefix.deepCopy()
	return prefix, nil
}
func (m *memory) DeletePrefix(_ context.Context, prefix Prefix, namespace string) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.prefixes[namespace]; !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}
	delete(m.prefixes[namespace], prefix.Cidr)
	return *prefix.deepCopy(), nil
}
