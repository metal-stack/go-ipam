package ipam

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type kubeConfigMap struct {
	client       client.Client
	configMapKey types.NamespacedName
	lock         sync.Mutex
}

func NewKubeConfigMap(ctx context.Context, client client.Client, namespacedName types.NamespacedName) (Storage, error) {
	kcm := &kubeConfigMap{
		client:       client,
		configMapKey: namespacedName,
	}

	if err := kcm.CreateNamespace(ctx, defaultNamespace); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	return kcm, nil
}

// loadConfigMap loads the configmap from the kubernetes API.
// If the configmap does not exist, it returns an empty configmap
// with the correct name and namespace.
func (k *kubeConfigMap) loadConfigMap(ctx context.Context) (corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}

	if err := k.client.Get(ctx, k.configMapKey, &cm); err != nil {
		if kerrors.IsNotFound(err) {
			cm.Name = k.configMapKey.Name
			cm.Namespace = k.configMapKey.Namespace
		} else {
			return cm, fmt.Errorf("get configmap: %w", err)
		}
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	return cm, nil
}

// storeConfigMap stores the configmap in the kubernetes API.
// If the configmap does not exist, it creates it.
func (k *kubeConfigMap) storeConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	if err := k.client.Update(ctx, cm); err != nil {
		if kerrors.IsNotFound(err) {
			if err := k.client.Create(ctx, cm); err != nil {
				return fmt.Errorf("create configmap: %w", err)
			}
		} else {
			return fmt.Errorf("update configmap: %w", err)
		}
	}

	return nil
}

func (k *kubeConfigMap) checkIpamNamespaceExists(cm *corev1.ConfigMap, namespace string) error {
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	if _, ok := cm.Data[namespace]; ok {
		return nil
	}

	return ErrNamespaceDoesNotExist
}

// CreateNamespace implements Storage.
func (k *kubeConfigMap) CreateNamespace(ctx context.Context, namespace string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("load configmap: %w", err)
	}

	if _, ok := cm.Data[namespace]; ok {
		return nil
	}

	cm.Data[namespace] = "{}"

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return fmt.Errorf("store configmap: %w", err)
	}

	return nil
}

// CreatePrefix implements Storage.
func (k *kubeConfigMap) CreatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return Prefix{}, fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return Prefix{}, fmt.Errorf("check ipam namespace exists: %w", err)
	}

	prefixMap := make(map[string]prefixJSON)
	if err := json.Unmarshal([]byte(cm.Data[namespace]), &prefixMap); err != nil {
		return Prefix{}, fmt.Errorf("unmarshal namespace: %w", err)
	}

	if _, ok := prefixMap[prefix.Cidr]; ok {
		return Prefix{}, ErrAlreadyAllocated
	}

	prefixMap[prefix.Cidr] = prefix.toPrefixJSON()

	marshalledPrefixes, err := json.Marshal(prefixMap)
	if err != nil {
		return Prefix{}, fmt.Errorf("marshal namespace: %w", err)
	}

	cm.Data[namespace] = string(marshalledPrefixes)

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return Prefix{}, fmt.Errorf("store configmap: %w", err)
	}

	return prefix, nil
}

// DeleteAllPrefixes implements Storage.
func (k *kubeConfigMap) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return fmt.Errorf("check ipam namespace exists: %w", err)
	}

	cm.Data[namespace] = "{}"

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return fmt.Errorf("store configmap: %w", err)
	}

	return nil
}

// DeleteNamespace implements Storage.
func (k *kubeConfigMap) DeleteNamespace(ctx context.Context, namespace string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("load configmap: %w", err)
	}

	delete(cm.Data, namespace)

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return fmt.Errorf("store configmap: %w", err)
	}

	return nil
}

// DeletePrefix implements Storage.
func (k *kubeConfigMap) DeletePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return Prefix{}, fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return Prefix{}, fmt.Errorf("check ipam namespace exists: %w", err)
	}

	prefixMap := make(map[string]Prefix)
	if err := json.Unmarshal([]byte(cm.Data[namespace]), &prefixMap); err != nil {
		return Prefix{}, fmt.Errorf("unmarshal namespace: %w", err)
	}

	if _, ok := prefixMap[prefix.Cidr]; !ok {
		return Prefix{}, ErrNotFound
	}

	delete(prefixMap, prefix.Cidr)

	marshalledPrefixes, err := json.Marshal(prefixMap)
	if err != nil {
		return Prefix{}, fmt.Errorf("marshal namespace: %w", err)
	}

	cm.Data[namespace] = string(marshalledPrefixes)

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return Prefix{}, fmt.Errorf("store configmap: %w", err)
	}

	return prefix, nil
}

// ListNamespaces implements Storage.
func (k *kubeConfigMap) ListNamespaces(ctx context.Context) ([]string, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("load configmap: %w", err)
	}

	namespaces := make([]string, 0, len(cm.Data))
	for namespace := range cm.Data {
		namespaces = append(namespaces, namespace)
	}

	return namespaces, nil
}

// Name implements Storage.
func (k *kubeConfigMap) Name() string {
	return "kube-configmap"
}

// ReadAllPrefixCidrs implements Storage.
func (k *kubeConfigMap) ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	prefixes, err := k.ReadAllPrefixes(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("read all prefixes: %w", err)
	}

	cidrs := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		cidrs = append(cidrs, prefix.Cidr)
	}

	return cidrs, nil
}

// ReadAllPrefixes implements Storage.
func (k *kubeConfigMap) ReadAllPrefixes(ctx context.Context, namespace string) (Prefixes, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return Prefixes{}, fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return Prefixes{}, fmt.Errorf("check ipam namespace exists: %w", err)
	}

	prefixMap := make(map[string]prefixJSON)
	if err := json.Unmarshal([]byte(cm.Data[namespace]), &prefixMap); err != nil {
		return Prefixes{}, fmt.Errorf("unmarshal namespace: %w", err)
	}

	prefixes := make(Prefixes, 0)
	for _, pfx := range prefixMap {
		prefixes = append(prefixes, pfx.toPrefix())
	}

	return prefixes, nil
}

// ReadPrefix implements Storage.
func (k *kubeConfigMap) ReadPrefix(ctx context.Context, prefix string, namespace string) (Prefix, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return Prefix{}, fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return Prefix{}, fmt.Errorf("check ipam namespace exists: %w", err)
	}

	prefixMap := make(map[string]prefixJSON)
	if err := json.Unmarshal([]byte(cm.Data[namespace]), &prefixMap); err != nil {
		return Prefix{}, fmt.Errorf("unmarshal namespace: %w", err)
	}

	pfx, ok := prefixMap[prefix]
	if !ok {
		return Prefix{}, fmt.Errorf("%w: prefix %v not found", ErrNotFound, prefix)
	}

	return pfx.toPrefix(), nil
}

// UpdatePrefix implements Storage.
func (k *kubeConfigMap) UpdatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	cm, err := k.loadConfigMap(ctx)
	if err != nil {
		return Prefix{}, fmt.Errorf("load configmap: %w", err)
	}

	if err := k.checkIpamNamespaceExists(&cm, namespace); err != nil {
		return Prefix{}, fmt.Errorf("check ipam namespace exists: %w", err)
	}

	prefixMap := make(map[string]prefixJSON)
	if err := json.Unmarshal([]byte(cm.Data[namespace]), &prefixMap); err != nil {
		return Prefix{}, fmt.Errorf("unmarshal namespace: %w", err)
	}

	if _, ok := prefixMap[prefix.Cidr]; !ok {
		return Prefix{}, ErrNotFound
	}

	storedPrefix := prefixMap[prefix.Cidr].toPrefix()

	if storedPrefix.version > prefix.version {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}

	prefix.version++
	prefixMap[prefix.Cidr] = prefix.toPrefixJSON()

	marshalledPrefixes, err := json.Marshal(prefixMap)
	if err != nil {
		return Prefix{}, fmt.Errorf("marshal namespace: %w", err)
	}

	cm.Data[namespace] = string(marshalledPrefixes)

	if err := k.storeConfigMap(ctx, &cm); err != nil {
		return Prefix{}, fmt.Errorf("store configmap: %w", err)
	}

	return prefix, nil
}
