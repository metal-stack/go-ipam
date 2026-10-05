// Package redis provides a Storage implementation backed by Redis (or
// Redis-compatible databases such as KeyDB).
package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"

	redigo "github.com/redis/go-redis/v9"

	ipam "github.com/metal-stack/go-ipam"
)

const namespaceKey = "namespaces"

type redis struct {
	rdb        *redigo.Client
	namespaces map[string]struct{}
	lock       sync.RWMutex
}

// NewRedis create a redis storage for ipam
func NewRedis(ctx context.Context, ip, port string) (ipam.Storage, error) {
	return newRedis(ctx, ip, port)
}

// New is an alias for NewRedis.
func New(ctx context.Context, ip, port string) (ipam.Storage, error) {
	return newRedis(ctx, ip, port)
}

func (r *redis) Name() string {
	return "redis"
}

func newRedis(ctx context.Context, ip, port string) (*redis, error) {
	rdb := redigo.NewClient(&redigo.Options{
		Addr:     fmt.Sprintf("%s:%s", ip, port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	r := &redis{
		rdb:        rdb,
		namespaces: make(map[string]struct{}),
		lock:       sync.RWMutex{},
	}
	if err := r.CreateNamespace(ctx, ipam.DefaultNamespace); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *redis) checkNamespaceExists(ctx context.Context, namespace string) error {
	if _, ok := r.namespaces[namespace]; ok {
		return nil
	}
	found, err := r.rdb.SIsMember(ctx, namespaceKey, namespace).Result()
	if err != nil {
		return fmt.Errorf("error checking namespace: %w", err)
	}
	if !found {
		return ipam.ErrNamespaceDoesNotExist
	}
	r.namespaces[namespace] = struct{}{}
	return nil
}

func (r *redis) CreatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	existing, err := r.rdb.HExists(ctx, namespace, prefix.Cidr).Result()
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	if existing {
		return ipam.Prefix{}, fmt.Errorf("prefix:%v already exists", prefix)
	}
	pfx, err := prefix.ToJSON()
	if err != nil {
		return ipam.Prefix{}, err
	}
	if err = r.rdb.HSet(ctx, namespace, prefix.Cidr, pfx).Err(); err != nil {
		return ipam.Prefix{}, err
	}
	return prefix, err
}

func (r *redis) ReadPrefix(ctx context.Context, prefix, namespace string) (ipam.Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	result, err := r.rdb.HGet(ctx, namespace, prefix).Result()
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("%w unable to read existing prefix:%v, error:%w", ipam.ErrNotFound, prefix, err)
	}
	return ipam.FromJSON([]byte(result))
}

func (r *redis) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}
	return r.rdb.Del(ctx, namespace).Err()
}

func (r *redis) ReadAllPrefixes(ctx context.Context, namespace string) (ipam.Prefixes, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	pfxs, err := r.rdb.HGetAll(ctx, namespace).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	result := ipam.Prefixes{}
	for _, pfx := range pfxs {
		pfx, err := ipam.FromJSON([]byte(pfx))
		if err != nil {
			return nil, err
		}
		result = append(result, pfx)
	}
	return result, nil
}

func (r *redis) ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	pfxs, err := r.rdb.HGetAll(ctx, namespace).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	ps := make([]string, 0, len(pfxs))
	for cidr := range pfxs {
		ps = append(ps, cidr)
	}
	return ps, nil
}

func (r *redis) UpdatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	oldVersion := prefix.IncrVersion()
	pn, err := prefix.ToJSON()
	if err != nil {
		return ipam.Prefix{}, err
	}

	txf := func(tx *redigo.Tx) error {
		// Get current value or zero.
		p, err := tx.HGet(ctx, namespace, prefix.Cidr).Result()
		if err != nil && !errors.Is(err, redigo.Nil) {
			return err
		}
		oldPrefix, err := ipam.FromJSON([]byte(p))
		if err != nil {
			return err
		}
		// Actual operation (local in optimistic lock).
		if oldPrefix.Version() != oldVersion {
			return fmt.Errorf("%w: unable to update prefix:%s", ipam.ErrOptimisticLockError, prefix.Cidr)
		}

		// Operation is committed only if the watched keys remain unchanged.
		_, err = tx.TxPipelined(ctx, func(pipe redigo.Pipeliner) error {
			pipe.HSet(ctx, namespace, prefix.Cidr, pn)
			return nil
		})
		return err
	}
	err = r.rdb.Watch(ctx, txf, namespace)
	if err != nil {
		return ipam.Prefix{}, err
	}

	return prefix, nil
}

func (r *redis) DeletePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	if err := r.rdb.HDel(ctx, namespace, prefix.Cidr).Err(); err != nil {
		return *prefix.DeepCopy(), err
	}
	return *prefix.DeepCopy(), nil
}

func (r *redis) CreateNamespace(ctx context.Context, namespace string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.namespaces[namespace]; ok {
		return nil
	}
	if err := r.rdb.SAdd(ctx, namespaceKey, namespace).Err(); err != nil {
		return err
	}
	r.namespaces[namespace] = struct{}{}

	return nil
}

func (r *redis) ListNamespaces(ctx context.Context) ([]string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rdb.SMembers(ctx, namespaceKey).Result()
}

func (r *redis) DeleteNamespace(ctx context.Context, namespace string) error {
	if err := r.DeleteAllPrefixes(ctx, namespace); err != nil {
		return err
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if err := r.rdb.SRem(ctx, namespaceKey, namespace).Err(); err != nil {
		return err
	}
	delete(r.namespaces, namespace)
	return nil
}
