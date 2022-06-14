package ipam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	redigo "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

const redisPrefix = "GOIPAM"

func key(prefix string) string {
	return strings.Join([]string{redisPrefix, prefix}, "_")
}

func prefixFromKey(key string) string {
	return strings.TrimPrefix(key, redisPrefix+"_")
}

type redis struct {
	rdb  *redigo.Client
	lock sync.RWMutex
}

// NewRedis create a redis storage for ipam
func NewRedis(ip, port string) Storage {
	return newRedis(ip, port)
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

func (r *redis) CreatePrefix(prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	existing, err := r.rdb.Exists(ctx, key(prefix.Cidr)).Result()
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
	err = r.rdb.Set(ctx, key(prefix.Cidr), pfx, 0).Err()
	return prefix, err
}
func (r *redis) ReadPrefix(prefix string) (Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	result, err := r.rdb.Get(ctx, key(prefix)).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	return fromJSON([]byte(result))
}
func (r *redis) ReadAllPrefixes() ([]Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	prefixes := r.rdb.Scan(ctx, 0, key("*"), 0).Iterator()

	result := []Prefix{}
	for prefixes.Next(ctx) {
		v, err := r.rdb.Get(ctx, prefixes.Val()).Bytes()
		if err != nil {
			return nil, err
		}
		prefix, err := fromJSON(v)
		if err != nil {
			return nil, err
		}
		result = append(result, prefix)
	}
	if err := prefixes.Err(); err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	return result, nil
}
func (r *redis) ReadAllPrefixCidrs() ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	prefixes := r.rdb.Scan(ctx, 0, key("*"), 0).Iterator()
	result := []string{}
	for prefixes.Next(ctx) {
		result = append(result, prefixFromKey(prefixes.Val()))
	}
	if err := prefixes.Err(); err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	return result, nil
}
func (r *redis) UpdatePrefix(prefix Prefix) (Prefix, error) {
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
		p, err := tx.Get(ctx, key(prefix.Cidr)).Result()
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
			pipe.Set(ctx, key(prefix.Cidr), pn, 0)
			return nil
		})
		return err
	}
	err = r.rdb.Watch(ctx, txf, key(prefix.Cidr))
	if err != nil {
		return Prefix{}, err
	}

	return prefix, nil
}
func (r *redis) DeletePrefix(prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, err := r.rdb.Del(ctx, key(prefix.Cidr)).Result()
	if err != nil {
		return *prefix.deepCopy(), err
	}
	return *prefix.deepCopy(), nil
}
