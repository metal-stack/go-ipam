package ipam

import (
	"context"
	"fmt"
	"sync"
	"time"

	redigo "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type redis struct {
	rdb  *redigo.Client
	lock sync.RWMutex
}

// NewRedis create a redis storage for ipam
func NewRedis(ip, port string) Storage {
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
func (r *redis) ReadPrefix(prefix string) (Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	result, err := r.rdb.Get(ctx, prefix).Result()
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}
	return fromJSON([]byte(result))
}
func (r *redis) ReadAllPrefixes() ([]Prefix, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	ss := r.rdb.Keys(ctx, "*")
	pfxs, err := ss.Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	result := []Prefix{}
	for _, pfx := range pfxs {
		s := r.rdb.Get(ctx, pfx)
		v, err := s.Bytes()
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
func (r *redis) ReadAllPrefixCidrs() ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	ss := r.rdb.Keys(ctx, "*")
	pfxs, err := ss.Result()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}
	return pfxs, nil
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

	// TODO add r.rdb.Multi aka transaktions

	oldPrefix, err := r.ReadPrefix(prefix.Cidr)
	if err != nil {
		return Prefix{}, err
	}
	if oldPrefix.version != oldVersion {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}

	s := r.rdb.Set(ctx, prefix.Cidr, pn, 1*time.Second)
	if s.Err() != nil {
		return Prefix{}, fmt.Errorf("%w: updatePrefix did not effect any row", ErrOptimisticLockError)
	}

	return prefix, nil
}
func (r *redis) DeletePrefix(prefix Prefix) (Prefix, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	res := r.rdb.Del(ctx, prefix.Cidr)
	if res.Err() != nil {
		return *prefix.deepCopy(), res.Err()
	}
	return *prefix.deepCopy(), nil
}
