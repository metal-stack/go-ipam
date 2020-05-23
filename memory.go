package ipam

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type memory struct {
	prefixes map[string]Prefix
	lock     sync.RWMutex
}

// NewMemory create a memory storage for ipam
func NewMemory() *memory {
	prefixes := make(map[string]Prefix)
	return &memory{
		prefixes: prefixes,
		lock:     sync.RWMutex{},
	}
}

func (m *memory) CreatePrefix(namespace *string, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := namespacedKey(namespace, prefix)
	_, ok := m.prefixes[key]
	if ok {
		return Prefix{}, fmt.Errorf("prefix already created:%v", prefix)
	}
	m.prefixes[key] = *prefix.DeepCopy()
	return prefix, nil
}
func (m *memory) ReadPrefix(namespace *string, prefix string) (Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	key := namespacePrefix(namespace) + prefix
	result, ok := m.prefixes[key]
	if !ok {
		return Prefix{}, errors.Errorf("Prefix %s not found", prefix)
	}
	return *result.DeepCopy(), nil
}
func (m *memory) ReadAllPrefixes(namespace *string) ([]Prefix, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	ps := make([]Prefix, 0, len(m.prefixes))
	for k, v := range m.prefixes {
		if namespace != nil && !strings.HasPrefix(k, namespacePrefix(namespace)) {
			continue
		}
		ps = append(ps, *v.DeepCopy())
	}
	return ps, nil
}
func (m *memory) UpdatePrefix(namespace *string, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if prefix.Cidr == "" {
		return Prefix{}, fmt.Errorf("prefix not present:%v", prefix)
	}

	key := namespacedKey(namespace, prefix)
	_, ok := m.prefixes[key]
	if !ok {
		return Prefix{}, fmt.Errorf("prefix not found:%s", prefix.Cidr)
	}
	m.prefixes[key] = *prefix.DeepCopy()
	return prefix, nil
}
func (m *memory) DeletePrefix(namespace *string, prefix Prefix) (Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	key := namespacedKey(namespace, prefix)
	delete(m.prefixes, key)
	return *prefix.DeepCopy(), nil
}
