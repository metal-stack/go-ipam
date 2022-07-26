package ipam

import (
	"context"
	"errors"
	"fmt"
	"sync"

	redigo "github.com/go-redis/redis/v8"
)

type redis struct {
	rdb  *redigo.Client
	lock sync.RWMutex
}

// NewRedis create a redis storage for ipam
func NewRedis(ip, port string) Storage {
	return newRedis(ip, port)
}
func (r *redis) Name() string {
	return "redis"
}

func newRedis(ip, port string) *redis {
	rdb := redigo.NewClient(&redigo.Options{
		Addr:     fmt.Sprintf("%s:%s", ip, port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &redis{
		rdb:  rdb,
		lock: sync.RWMutex{},
	}
}

func (r *redis) CreatePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	existing, err := r.rdb.Exists(ctx, prefix.Cidr).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	if existing != 0 {
		return Prefix{}, fmt.Errorf("prefix:%v already exists", prefix)
	}
	pfx, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}
	err = r.rdb.Set(ctx, prefix.Cidr, pfx, 0).Err()
	return prefix, err
}
func (r *redis) ReadPrefix(ctx context.Context, prefix string) (Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	result, err := r.rdb.Get(ctx, prefix).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	return fromJSON([]byte(result))
}
func (r *redis) DeleteAllPrefixes(ctx context.Context) error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, err := r.rdb.FlushAll(ctx).Result()
	return err
}
func (r *redis) ReadAllPrefixes(ctx context.Context) (Prefixes, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	pfxs, err := r.rdb.Keys(ctx, "*").Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	result := Prefixes{}
	for _, pfx := range pfxs {
		v, err := r.rdb.Get(ctx, pfx).Bytes()
		if err != nil {
			return nil, err
		}
		pfx, err := fromJSON(v)
		if err != nil {
			return nil, err
		}
		result = append(result, pfx)
	}
	return result, nil
}
func (r *redis) ReadAllPrefixCidrs(ctx context.Context) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	pfxs, err := r.rdb.Keys(ctx, "*").Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	return pfxs, nil
}
func (r *redis) UpdatePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	pn, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}

	txf := func(tx *redigo.Tx) error {
		// Get current value or zero.
		p, err := tx.Get(ctx, prefix.Cidr).Result()
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
			pipe.Set(ctx, prefix.Cidr, pn, 0)
			return nil
		})
		return err
	}
	err = r.rdb.Watch(ctx, txf, prefix.Cidr)
	if err != nil {
		return Prefix{}, err
	}

	return prefix, nil
}
func (r *redis) DeletePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, err := r.rdb.Del(ctx, prefix.Cidr).Result()
	if err != nil {
		return *prefix.deepCopy(), err
	}
	return *prefix.deepCopy(), nil
}
