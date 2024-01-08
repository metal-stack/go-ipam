package ipam

import (
	"context"
	"errors"
	"fmt"
	"sync"

	redigo "github.com/redis/go-redis/v9"
)

const namespaceKey = "namespaces"

type redis struct {
	rdb        *redigo.Client
	namespaces map[string]struct{}
	lock       sync.RWMutex
}

// NewRedis create a redis storage for ipam
func NewRedis(ctx context.Context, ip, port string) (Storage, error) {
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
	if err := r.CreateNamespace(ctx, defaultNamespace); err != nil {
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
		return ErrNamespaceDoesNotExist
	}
	r.namespaces[namespace] = struct{}{}
	return nil
}

func (r *redis) CreatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}

	existing, err := r.rdb.HExists(ctx, namespace, prefix.Cidr).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	if existing {
		return Prefix{}, fmt.Errorf("prefix:%v already exists", prefix)
	}
	pfx, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}
	if err = r.rdb.HSet(ctx, namespace, prefix.Cidr, pfx).Err(); err != nil {
		return Prefix{}, err
	}
	return prefix, err
}
func (r *redis) ReadPrefix(ctx context.Context, prefix, namespace string) (Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}

	result, err := r.rdb.HGet(ctx, namespace, prefix).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("%w unable to read existing prefix:%v, error:%w", ErrNotFound, prefix, err)
	}
	return fromJSON([]byte(result))
}

func (r *redis) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}
	return r.rdb.Del(ctx, namespace).Err()
}

func (r *redis) ReadAllPrefixes(ctx context.Context, namespace string) (Prefixes, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	pfxs, err := r.rdb.HGetAll(ctx, namespace).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	result := Prefixes{}
	for _, pfx := range pfxs {
		pfx, err := fromJSON([]byte(pfx))
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
func (r *redis) UpdatePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}

	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}

	txf := func(tx *redigo.Tx) error {
		// Get current value or zero.
		p, err := tx.HGet(ctx, namespace, prefix.Cidr).Result()
		if err != nil && !errors.Is(err, redigo.Nil) {
			return err
		}
		oldPrefix, err := fromJSON([]byte(p))
		if err != nil {
			return err
		}
		// Actual operation (local in optimistic lock).
		if oldPrefix.version != oldVersion {
			return fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
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
		return Prefix{}, err
	}

	return prefix, nil
}
func (r *redis) DeletePrefix(ctx context.Context, prefix Prefix, namespace string) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if err := r.checkNamespaceExists(ctx, namespace); err != nil {
		return Prefix{}, err
	}

	if err := r.rdb.HDel(ctx, namespace, prefix.Cidr).Err(); err != nil {
		return *prefix.deepCopy(), err
	}
	return *prefix.deepCopy(), nil
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
