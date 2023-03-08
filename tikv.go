package ipam

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"time"

	tikverr "github.com/tikv/client-go/v2/error"
	"github.com/tikv/client-go/v2/kv"
	"github.com/tikv/client-go/v2/txnkv"
)

type tikv struct {
	client *txnkv.Client
}

// NewTikv create a tikv storage for ipam
func NewTikv(addrs []netip.AddrPort) Storage {
	return newTikv(addrs)
}

func (t *tikv) Name() string {
	return "tikv"
}

func newTikv(addrs []netip.AddrPort) *tikv {
	tikvaddrs := []string{}
	for _, addr := range addrs {
		tikvaddrs = append(tikvaddrs, addr.String())
	}

	client, err := txnkv.NewClient(tikvaddrs)
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

	existing, err := txn.Get(ctx, primaryKey)
	if err != nil && !errors.Is(err, tikverr.ErrNotExist) {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	if existing != nil {
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
	if err != nil && !errors.Is(err, tikverr.ErrNotExist) {
		return Prefix{}, fmt.Errorf("unable to read data from ETCD error:%w", err)
	}

	if get == nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v", prefix)
	}

	err = txn.Commit(ctx)
	if err != nil {
		return Prefix{}, err
	}
	return fromJSON(get)
}

func (t *tikv) DeleteAllPrefixes(ctx context.Context) error {
	txn := t.begin_pessimistic_txn()
	iter, err := txn.Iter([]byte(""), nil)
	if err != nil {
		return err
	}
	for iter.Valid() {
		err := txn.Delete(iter.Key())
		if err != nil {
			return err
		}
		err = iter.Next()
		if err != nil {
			return err
		}
	}

	err = txn.Commit(ctx)
	return err
}

func (t *tikv) ReadAllPrefixes(ctx context.Context) (Prefixes, error) {
	txn := t.begin_pessimistic_txn()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result := Prefixes{}
	iter, err := txn.Iter([]byte(""), nil)
	if err != nil {
		return nil, err
	}
	for iter.Valid() {
		pfx, err := fromJSON(iter.Value())
		if err != nil {
			return nil, err
		}
		result = append(result, pfx)
		err = iter.Next()
		if err != nil {
			return nil, err
		}
	}

	err = txn.Commit(ctx)
	return result, err
}
func (t *tikv) ReadAllPrefixCidrs(ctx context.Context) ([]string, error) {
	txn := t.begin_pessimistic_txn()

	iter, err := txn.Iter([]byte(""), nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	allPrefix := []string{}
	for iter.Valid() {
		cidr := string(iter.Key())
		allPrefix = append(allPrefix, cidr)
		err = iter.Next()
		if err != nil {
			return nil, err
		}
	}

	err = txn.Commit(ctx)
	return allPrefix, err
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
	if err != nil && !errors.Is(err, tikverr.ErrNotExist) {
		return Prefix{}, fmt.Errorf("unable to read data from tikv error:%w", err)
	}

	if get == nil {
		return Prefix{}, fmt.Errorf("unable to get prefix cidrs:%s", &prefix)
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
	err = txn.Commit(ctx)
	return prefix, err
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
	err = txn.Commit(ctx)
	return *prefix.deepCopy(), err
}
