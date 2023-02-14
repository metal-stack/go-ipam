package ipam

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tikv/client-go/v2/kv"
	"github.com/tikv/client-go/v2/txnkv"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type tikv struct {
	client *txnkv.Client
	lock   sync.RWMutex
}

// NewEtcd create a etcd storage for ipam
func NewTikv(ip, port string, cert, key []byte, insecureskip bool) Storage {
	return newEtcd(ip, port, cert, key, insecureskip)
}

func (t *tikv) Name() string {
	return "etcd"
}

func newTikv(ip, port string) *tikv {
	pdAddr := fmt.Sprintf("%s:%s", ip, port)
	client, err := txnkv.NewClient([]string{pdAddr})
	if err != nil {
		log.Fatal(err)
	}
	return &tikv{
		client: client,
	}
}
func (t *tikv) begin_pessimistic_txn() (txn *txnkv.KVTxn) {
	txn, err := t.client.Begin()
	if err != nil {
		panic(err)
	}
	txn.SetPessimistic(true)
	return txn
}

func (t *tikv) CreatePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {

	txn := t.begin_pessimistic_txn()
	primaryKey := []byte(prefix.Cidr)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// txn: lock the primary key
	err := txn.LockKeysWithWaitTime(ctx, kv.LockAlwaysWait, primaryKey)
	if err != nil {
		return Prefix{}, err
	}

	_, err = txn.Get(ctx, primaryKey)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	if txn.Len() != 0 {
		return Prefix{}, fmt.Errorf("prefix already exists:%v", prefix)
	}

	pfx, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = txn.Set(primaryKey, pfx)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to create prefix:%v, error:%w", prefix, err)
	}

	err = txn.Commit(ctx)

	return prefix, err
}

func (t *tikv) ReadPrefix(ctx context.Context, prefix string) (Prefix, error) {
	txn := t.begin_pessimistic_txn()
	primaryKey := []byte(prefix)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// txn: lock the primary key
	err := txn.LockKeysWithWaitTime(ctx, kv.LockAlwaysWait, primaryKey)
	if err != nil {
		return Prefix{}, err
	}

	get, err := txn.Get(ctx, primaryKey)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read data from ETCD error:%w", err)
	}

	if txn.Len() != 0 {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	err = txn.Commit(ctx)
	if err != nil {
		return Prefix{}, err
	}
	return fromJSON(get)
}

func (t *tikv) DeleteAllPrefixes(ctx context.Context) error {
	// txn := t.begin_pessimistic_txn()
	ctx, cancel := context.WithTimeout(ctx, 50*time.Minute)
	defer cancel()

	// txn.Iter()
	// defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	// pfxs, err := e.etcdDB.Get(ctx, "", defaultOpts...)
	// defer cancel()
	// if err != nil {
	// 	return fmt.Errorf("unable to get all prefix cidrs:%w", err)
	// }

	// for _, pfx := range pfxs.Kvs {
	// 	_, err := e.etcdDB.Delete(ctx, string(pfx.Key))
	// 	if err != nil {
	// 		return fmt.Errorf("unable to delete prefix:%w", err)
	// 	}
	// }
	return nil
}

func (t *tikv) ReadAllPrefixes(ctx context.Context) (Prefixes, error) {
	// e.lock.Lock()
	// defer e.lock.Unlock()

	// ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	// pfxs, err := e.etcdDB.Get(ctx, "", defaultOpts...)
	// defer cancel()
	// if err != nil {
	// 	return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	// }

	result := Prefixes{}
	// for _, pfx := range pfxs.Kvs {
	// 	v, err := e.etcdDB.Get(ctx, string(pfx.Key))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	pfx, err := fromJSON(v.Kvs[0].Value)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	result = append(result, pfx)
	// }
	return result, nil
}
func (t *tikv) ReadAllPrefixCidrs(ctx context.Context) ([]string, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	allPrefix := []string{}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defaultOpts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSerializable()}
	pfxs, err := e.etcdDB.Get(ctx, "", defaultOpts...)
	defer cancel()
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	for _, pfx := range pfxs.Kvs {
		allPrefix = append(allPrefix, string(pfx.Key))
	}

	return allPrefix, nil
}
func (t *tikv) UpdatePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {
	txn := t.begin_pessimistic_txn()
	primaryKey := []byte(prefix.Cidr)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// txn: lock the primary key
	err := txn.LockKeysWithWaitTime(ctx, kv.LockAlwaysWait, primaryKey)
	if err != nil {
		return Prefix{}, err
	}

	oldVersion := prefix.version
	prefix.version = oldVersion + 1
	newPrefix, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}

	get, err := txn.Get(ctx, primaryKey)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read data from tikv error:%w", err)
	}

	if txn.Len() == 0 {
		return Prefix{}, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	oldPrefix, err := fromJSON(get)
	if err != nil {
		return Prefix{}, err
	}

	// Actual operation (local in optimistic lock).
	if oldPrefix.version != oldVersion {
		return Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ErrOptimisticLockError, prefix.Cidr)
	}

	err = txn.Set(primaryKey, []byte(newPrefix))
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to update prefix:%s, error:%w", prefix.Cidr, err)
	}

	return prefix, nil
}
func (t *tikv) DeletePrefix(ctx context.Context, prefix Prefix) (Prefix, error) {
	txn := t.begin_pessimistic_txn()
	primaryKey := []byte(prefix.Cidr)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// txn: lock the primary key
	err := txn.LockKeysWithWaitTime(ctx, kv.LockAlwaysWait, primaryKey)
	if err != nil {
		return Prefix{}, err
	}

	err = txn.Delete(primaryKey)
	if err != nil {
		return *prefix.deepCopy(), err
	}
	return *prefix.deepCopy(), nil
}
