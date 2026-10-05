// Package etcd provides a Storage implementation backed by etcd.
package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	ipam "github.com/metal-stack/go-ipam"
)

const namespaceKey = "namespaces"

type etcd struct {
	etcdDB     *clientv3.Client
	namespaces map[string]struct{}
	lock       sync.RWMutex
}

// NewEtcd create a etcd storage for ipam
func NewEtcd(ctx context.Context, ip, port string, cert, key []byte, insecureskip bool) (ipam.Storage, error) {
	return newEtcd(ctx, ip, port, cert, key, insecureskip)
}

// New is an alias for NewEtcd.
func New(ctx context.Context, ip, port string, cert, key []byte, insecureskip bool) (ipam.Storage, error) {
	return newEtcd(ctx, ip, port, cert, key, insecureskip)
}

func (e *etcd) Name() string {
	return "etcd"
}

func newEtcd(ctx context.Context, ip, port string, cert, key []byte, insecureskip bool) (*etcd, error) {
	etcdConfig := clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("%s:%s", ip, port)},
		DialTimeout: 5 * time.Second,
		Context:     context.Background(),
	}

	if cert != nil && key != nil {
		// SSL
		clientCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			log.Fatal(err)
		}
		tls := &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			// nolint:gosec
			// #nosec G402
			InsecureSkipVerify: insecureskip,
		}
		etcdConfig.TLS = tls
	}
	cli, err := clientv3.New(etcdConfig)
	if err != nil {
		log.Fatal(err)
	}

	e := &etcd{
		etcdDB:     cli,
		namespaces: make(map[string]struct{}),
		lock:       sync.RWMutex{},
	}

	if err := e.CreateNamespace(ctx, ipam.DefaultNamespace); err != nil {
		return nil, err
	}

	return e, nil
}

func etcdNamespaceKey(namespace string) string {
	return namespaceKey + "/" + namespace
}

// This should ONLY be called when e.Lock() has been acquired
func (e *etcd) checkNamespaceExists(ctx context.Context, namespace string) error {
	if _, ok := e.namespaces[namespace]; ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := e.etcdDB.Get(ctx, etcdNamespaceKey(namespace))
	if err != nil {
		return fmt.Errorf("unable to read namespace key: %w", err)
	}
	if res.Count == 0 {
		return ipam.ErrNamespaceDoesNotExist
	}
	e.namespaces[namespace] = struct{}{}
	return nil
}

func (e *etcd) CreatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := namespace + "@" + prefix.Cidr
	get, err := e.etcdDB.Get(ctx, key)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	if get.Count != 0 {
		return ipam.Prefix{}, fmt.Errorf("prefix already exists:%v", prefix)
	}

	pfx, err := prefix.ToJSON()
	if err != nil {
		return ipam.Prefix{}, err
	}
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = e.etcdDB.Put(ctx, key, string(pfx))
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to create prefix:%v, error:%w", prefix, err)
	}

	return prefix, nil
}

func (e *etcd) ReadPrefix(ctx context.Context, prefix string, namespace string) (ipam.Prefix, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := namespace + "@" + prefix
	get, err := e.etcdDB.Get(ctx, key)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read data from ETCD error:%w", err)
	}

	if get.Count == 0 {
		return ipam.Prefix{}, fmt.Errorf("%w unable to read existing prefix:%v, error:%w", ipam.ErrNotFound, prefix, err)
	}

	return ipam.FromJSON(get.Kvs[0].Value)
}

func (e *etcd) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	e.lock.RLock()
	defer e.lock.RUnlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 50*time.Minute)
	defer cancel()
	defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	pfxs, err := e.etcdDB.Get(ctx, namespace, defaultOpts...)
	if err != nil {
		return fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	for _, pfx := range pfxs.Kvs {
		_, err := e.etcdDB.Delete(ctx, string(pfx.Key))
		if err != nil {
			return fmt.Errorf("unable to delete prefix:%w", err)
		}
	}
	return nil
}

func (e *etcd) ReadAllPrefixes(ctx context.Context, namespace string) (ipam.Prefixes, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	pfxs, err := e.etcdDB.Get(ctx, namespace, defaultOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	result := ipam.Prefixes{}
	for _, pfx := range pfxs.Kvs {
		v, err := e.etcdDB.Get(ctx, string(pfx.Key))
		if err != nil {
			return nil, err
		}
		pfx, err := ipam.FromJSON(v.Kvs[0].Value)
		if err != nil {
			return nil, err
		}
		result = append(result, pfx)
	}
	return result, nil
}

func (e *etcd) ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	allPrefix := []string{}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	pfxs, err := e.etcdDB.Get(ctx, namespace, defaultOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	for _, pfx := range pfxs.Kvs {
		v, err := e.etcdDB.Get(ctx, string(pfx.Key))
		if err != nil {
			return nil, err
		}
		pfx, err := ipam.FromJSON(v.Kvs[0].Value)
		if err != nil {
			return nil, err
		}
		allPrefix = append(allPrefix, string(pfx.Cidr))
	}

	return allPrefix, nil
}

func (e *etcd) UpdatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	oldVersion := prefix.IncrVersion()
	pn, err := prefix.ToJSON()
	if err != nil {
		return ipam.Prefix{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := namespace + "@" + prefix.Cidr
	p, err := e.etcdDB.Get(ctx, key)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read cidrs from ETCD:%w", err)
	}

	if p.Count == 0 {
		return ipam.Prefix{}, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	oldPrefix, err := ipam.FromJSON([]byte(p.Kvs[0].Value))
	if err != nil {
		return ipam.Prefix{}, err
	}

	// Actual operation (local in optimistic lock).
	if oldPrefix.Version() != oldVersion {
		return ipam.Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ipam.ErrOptimisticLockError, prefix.Cidr)
	}

	// Operation is committed only if the watched keys remain unchanged.
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = e.etcdDB.Put(ctx, key, string(pn))
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to update prefix:%s, error:%w", prefix.Cidr, err)
	}

	return prefix, nil
}

func (e *etcd) DeletePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := e.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	key := namespace + "@" + prefix.Cidr
	_, err := e.etcdDB.Delete(ctx, key)
	if err != nil {
		return *prefix.DeepCopy(), err
	}
	return *prefix.DeepCopy(), nil
}

func (e *etcd) CreateNamespace(ctx context.Context, namespace string) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	if _, ok := e.namespaces[namespace]; ok {
		return nil
	}
	key := etcdNamespaceKey(namespace)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := e.etcdDB.Put(ctx, key, "")
	e.namespaces[namespace] = struct{}{}
	return err
}

func (e *etcd) ListNamespaces(ctx context.Context) ([]string, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	key := etcdNamespaceKey("")
	defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, err := e.etcdDB.Get(ctx, key, defaultOpts...)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, kv := range res.Kvs {
		result = append(result, strings.TrimPrefix(string(kv.Key), key))
	}
	return result, nil
}

func (e *etcd) DeleteNamespace(ctx context.Context, namespace string) error {
	if err := e.DeleteAllPrefixes(ctx, namespace); err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	_, err := e.etcdDB.Delete(ctx, etcdNamespaceKey(namespace))
	delete(e.namespaces, namespace)
	return err
}
