// Package mongodb provides a Storage implementation backed by MongoDB.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	ipam "github.com/metal-stack/go-ipam"
)

const dbCidr = `prefix.cidr`
const versionKey = `version`

// MongoConfig configures the mongodb backed Storage.
type MongoConfig struct {
	DatabaseName       string
	MongoClientOptions *options.ClientOptions
}

type mongodb struct {
	db         *mongo.Database
	namespaces map[string]struct{}
	lock       sync.RWMutex
}

// New creates a mongodb storage for ipam.
func New(ctx context.Context, config MongoConfig) (ipam.Storage, error) {
	return newMongo(ctx, config)
}

func (m *mongodb) Name() string {
	return "mongodb"
}

func newMongo(ctx context.Context, config MongoConfig) (*mongodb, error) {
	m, err := mongo.Connect(ctx, config.MongoClientOptions)
	if err != nil {
		return nil, err
	}

	err = m.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := &mongodb{
		db:         m.Database(config.DatabaseName),
		namespaces: make(map[string]struct{}),
		lock:       sync.RWMutex{},
	}
	if err := db.CreateNamespace(ctx, ipam.DefaultNamespace); err != nil {
		return nil, err
	}
	return db, nil
}

func (m *mongodb) checkNamespaceExists(ctx context.Context, namespace string) error {
	if _, ok := m.namespaces[namespace]; ok {
		return nil
	}

	r, err := m.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return ipam.ErrNotFound
	}

	for _, ns := range r {
		m.namespaces[ns] = struct{}{}
	}

	if _, ok := m.namespaces[namespace]; !ok {
		return ipam.ErrNamespaceDoesNotExist
	}

	return nil
}

func (m *mongodb) CreatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	f := bson.D{{Key: dbCidr, Value: prefix.Cidr}}
	r := m.db.Collection(namespace).FindOne(ctx, f)

	// ErrNoDocuments should be returned if the prefix does not exist
	if r.Err() == nil {
		return ipam.Prefix{}, fmt.Errorf("prefix already exists:%s", prefix.Cidr)
	} else if r.Err() != nil && !errors.Is(r.Err(), mongo.ErrNoDocuments) { // unrelated to ErrNoDocuments.
		return ipam.Prefix{}, fmt.Errorf("unable to insert prefix:%s, error:%w", prefix.Cidr, r.Err())
	} // ErrNoDocuments should pass through this block

	_, err := m.db.Collection(namespace).InsertOne(ctx, prefix.ToPrefixJSON())
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to insert prefix:%s, error:%w", prefix.Cidr, err)
	}

	return prefix, nil
}

func (m *mongodb) ReadPrefix(ctx context.Context, prefix string, namespace string) (ipam.Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	f := bson.D{{Key: dbCidr, Value: prefix}}
	r := m.db.Collection(namespace).FindOne(ctx, f)

	// ErrNoDocuments should be returned if the prefix does not exist
	if r.Err() != nil && errors.Is(r.Err(), mongo.ErrNoDocuments) {
		return ipam.Prefix{}, fmt.Errorf(`%w prefix not found:%s, error:%w`, ipam.ErrNotFound, prefix, r.Err())
	} else if r.Err() != nil {
		return ipam.Prefix{}, fmt.Errorf(`error while trying to find prefix:%s, error:%w`, prefix, r.Err())
	}

	j := ipam.PrefixJSON{}
	err := r.Decode(&j)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read prefix:%w", err)
	}
	return j.ToPrefix(), nil
}

func (m *mongodb) DeleteAllPrefixes(ctx context.Context, namespace string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}

	f := bson.D{}
	_, err := m.db.Collection(namespace).DeleteMany(ctx, f)
	if err != nil {
		return fmt.Errorf(`error deleting all prefixes: %w`, err)
	}
	return nil
}

func (m *mongodb) ReadAllPrefixes(ctx context.Context, namespace string) (ipam.Prefixes, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	f := bson.D{} // match all documents
	c, err := m.db.Collection(namespace).Find(ctx, f)
	if err != nil {
		return nil, fmt.Errorf(`error reading all prefixes: %w`, err)
	}
	var r []ipam.PrefixJSON
	if err := c.All(ctx, &r); err != nil {
		return nil, fmt.Errorf(`error reading all prefixes: %w`, err)
	}

	var s = make(ipam.Prefixes, len(r))
	for i, v := range r {
		s[i] = v.ToPrefix()
	}

	return s, nil
}

func (m *mongodb) ReadPrefixes(ctx context.Context, namespace string) (ipam.Prefixes, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return nil, err
	}

	f := bson.D{} // match all documents
	c, err := m.db.Collection(namespace).Find(ctx, f)
	if err != nil {
		return nil, fmt.Errorf(`error reading all prefixes: %w`, err)
	}
	var r []ipam.PrefixJSON
	if err := c.All(ctx, &r); err != nil {
		return nil, fmt.Errorf(`error reading all prefixes: %w`, err)
	}

	var s = make(ipam.Prefixes, len(r))
	for i, v := range r {
		s[i] = v.ToPrefix()
	}

	return s, nil
}

func (m *mongodb) ReadAllPrefixCidrs(ctx context.Context, namespace string) ([]string, error) {
	p, err := m.ReadAllPrefixes(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var s = make([]string, len(p))
	for i, v := range p {
		s[i] = v.Cidr
	}
	return s, nil
}

func (m *mongodb) UpdatePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	oldVersion := prefix.IncrVersion()

	f := bson.D{{Key: dbCidr, Value: prefix.Cidr}, {Key: versionKey, Value: oldVersion}}

	o := options.Replace().SetUpsert(false)
	r, err := m.db.Collection(namespace).ReplaceOne(ctx, f, prefix.ToPrefixJSON(), o)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to update prefix:%s, error: %w", prefix.Cidr, err)
	}
	if r.MatchedCount == 0 {
		return ipam.Prefix{}, fmt.Errorf("%w: unable to update prefix:%s", ipam.ErrOptimisticLockError, prefix.Cidr)
	}
	if r.ModifiedCount == 0 {
		return ipam.Prefix{}, fmt.Errorf("%w: update did not effect any document:%s",
			ipam.ErrOptimisticLockError, prefix.Cidr)
	}

	return prefix, nil
}

func (m *mongodb) DeletePrefix(ctx context.Context, prefix ipam.Prefix, namespace string) (ipam.Prefix, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return ipam.Prefix{}, err
	}

	f := bson.D{{Key: dbCidr, Value: prefix.Cidr}}
	r := m.db.Collection(namespace).FindOneAndDelete(ctx, f)

	// ErrNoDocuments should be returned if the prefix does not exist
	if r.Err() != nil && errors.Is(r.Err(), mongo.ErrNoDocuments) {
		return ipam.Prefix{}, fmt.Errorf(`prefix not found:%s, error:%w`, prefix.Cidr, r.Err())
	} else if r.Err() != nil {
		return ipam.Prefix{}, fmt.Errorf(`error while trying to find prefix:%s, error:%w`, prefix.Cidr, r.Err())
	}

	j := ipam.PrefixJSON{}
	err := r.Decode(&j)
	if err != nil {
		return ipam.Prefix{}, fmt.Errorf("unable to read prefix:%w", err)
	}
	return j.ToPrefix(), nil
}

func (m *mongodb) CreateNamespace(ctx context.Context, namespace string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.namespaces[namespace]; ok {
		return nil
	}

	if err := m.db.CreateCollection(ctx, namespace); err != nil {
		var e mongo.CommandError
		if errors.As(err, &e) && e.Name == "NamespaceExists" {
			return nil
		}
		return err
	}
	_, err := m.db.Collection(namespace).Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: dbCidr, Value: 1}},
		Options: options.Index().SetUnique(true),
	}})
	m.namespaces[namespace] = struct{}{}
	return err
}

func (m *mongodb) ListNamespaces(ctx context.Context) ([]string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	r, err := m.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	// update our cache
	for _, ns := range r {
		m.namespaces[ns] = struct{}{}
	}
	return r, nil
}

func (m *mongodb) DeleteNamespace(ctx context.Context, namespace string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if err := m.checkNamespaceExists(ctx, namespace); err != nil {
		return err
	}
	return m.db.Collection(namespace).Drop(ctx)
}
