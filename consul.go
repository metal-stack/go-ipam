package ipam

import (
	"fmt"
	"sync"
	"net"
	"net/url"
	"strings"

	consulApi "github.com/hashicorp/consul/api"
)

type consul struct {
	kv *consulApi.KV
	keyPrefix string
	lock sync.RWMutex
}

// NewConsul create a consul storage for ipam
func NewConsul(consulUrl string, keyPrefix string) (Storage, error) {
	return newConsul(consulUrl, keyPrefix)
}

func newConsul(consulUrl string, keyPrefix string) (*consul, error) {
	var cfg *consulApi.Config

	if consulUrl == "" {
		cfg = consulApi.DefaultConfig()
	} else {
		u, err := url.Parse(consulUrl)
		if err != nil {
			return nil, err
		}

		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return nil, err
		}

		cfg = &consulApi.Config{
			Address: fmt.Sprintf("%s:%s", host, port),
			Scheme: u.Scheme,
		}
	}

	c, err := consulApi.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	var kp string = "go-ipam"
	if keyPrefix != "" {
		kp = keyPrefix
	}

	o := &consul{
		kv:  c.KV(),
		keyPrefix: kp,
		lock: sync.RWMutex{},
	}

	return o, nil
}

func (cl *consul) getKeyPath(key string) string {
	if key == "" {
		return fmt.Sprintf("%s/", strings.TrimSuffix(cl.keyPrefix, "/"))
	} else {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(cl.keyPrefix, "/"), key)
	}
}

func (cl *consul) CreatePrefix(prefix Prefix) (Prefix, error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	key := cl.getKeyPath(prefix.Cidr)

	existing, _, err := cl.kv.Get(key, nil)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	if existing != nil {
		return Prefix{}, fmt.Errorf("prefix:%v already exists", prefix)
	}

	pfx, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}

	kp := &consulApi.KVPair{Key: key, Value: pfx}
	_, err = cl.kv.Put(kp, nil)

	return prefix, err
}

func (cl *consul) ReadPrefix(prefix string) (Prefix, error) {
	cl.lock.RLock()
	defer cl.lock.RUnlock()

	key := cl.getKeyPath(prefix)

	result, _, err := cl.kv.Get(key, nil)
	if err != nil {
		return Prefix{}, fmt.Errorf("unable to read existing prefix:%v, error:%w", prefix, err)
	}

	if result == nil {
		return Prefix{}, fmt.Errorf("prefix:%v does not exist", prefix)
	}

	return fromJSON([]byte(result.Value))
}

func (cl *consul) ReadAllPrefixes() ([]Prefix, error) {
	cl.lock.RLock()
	defer cl.lock.RUnlock()

	key := cl.getKeyPath("")

	pfxs, _, err := cl.kv.Keys(key, "", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	result := []Prefix{}
	for _, pfx := range pfxs {
		key := cl.getKeyPath(pfx)

		kp, _, err := cl.kv.Get(key, nil)
		if err != nil {
			return nil, err
		}

		pfx, err := fromJSON(kp.Value)
		if err != nil {
			return nil, err
		}

		result = append(result, pfx)
	}

	return result, nil
}

func (cl *consul) ReadAllPrefixCidrs() ([]string, error) {
	cl.lock.RLock()
	defer cl.lock.RUnlock()

	key := cl.getKeyPath("")

	pfxs, _, err := cl.kv.Keys(key, "", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get all prefix cidrs:%w", err)
	}

	return pfxs, nil
}

func (cl *consul) UpdatePrefix(prefix Prefix) (Prefix, error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	oldVersion := prefix.version
	prefix.version = oldVersion + 1

	pn, err := prefix.toJSON()
	if err != nil {
		return Prefix{}, err
	}

	key := cl.getKeyPath(prefix.Cidr)
	kp := &consulApi.KVPair{Key: key, Value: pn}

	_, err = cl.kv.Put(kp, nil)
	return prefix, err
}

func (cl *consul) DeletePrefix(prefix Prefix) (Prefix, error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()

	key := cl.getKeyPath(prefix.Cidr)

	_, err := cl.kv.Delete(key, nil)
	return *prefix.deepCopy(), err
}
