package ipam

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	pgOnce           sync.Once
	pgContainer      testcontainers.Container
	pgVersion        string
	crOnce           sync.Once
	crContainer      testcontainers.Container
	cockroachVersion string
	redisOnce        sync.Once
	redisContainer   testcontainers.Container
	redisVersion     string
	keyDBOnce        sync.Once
	keyDBVersion     string
	keyDBContainer   testcontainers.Container
	etcdContainer    testcontainers.Container
	etcdVersion      string
	etcdOnce         sync.Once
	mdbOnce          sync.Once
	mdbContainer     testcontainers.Container
	mdbVersion       string

	backend string
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	pgVersion = os.Getenv("PG_VERSION")
	if pgVersion == "" {
		pgVersion = "17-alpine"
	}
	cockroachVersion = os.Getenv("COCKROACH_VERSION")
	if cockroachVersion == "" {
		cockroachVersion = "latest-v24.3"
	}
	redisVersion = os.Getenv("REDIS_VERSION")
	if redisVersion == "" {
		redisVersion = "7.4-alpine"
	}
	keyDBVersion = os.Getenv("KEYDB_VERSION")
	if keyDBVersion == "" {
		keyDBVersion = "latest"
	}
	etcdVersion = os.Getenv("ETCD_VERSION")
	if etcdVersion == "" {
		etcdVersion = "v3.5.19"
	}
	mdbVersion = os.Getenv("MONGODB_VERSION")
	if mdbVersion == "" {
		mdbVersion = "7"
	}
	backend = os.Getenv("BACKEND")
	if backend == "" {
		fmt.Printf("Using postgres:%s cockroach:%s redis:%s keydb:%s etcd:%s mongodb:%s\n", pgVersion, cockroachVersion, redisVersion, keyDBVersion, etcdVersion, mdbVersion)
	} else {
		fmt.Printf("only test %s\n", backend)
	}
	os.Exit(m.Run())
}

func startPostgres() (container testcontainers.Container, db *sql, err error) {
	ctx := context.Background()
	pgOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "postgres:" + pgVersion,
			ExposedPorts: []string{"5432/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
			Tmpfs:        map[string]string{"/var/lib/postgresql/data": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			),
			Cmd: []string{"postgres", "-c", "max_connections=200"},
		}
		pgContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := pgContainer.Host(ctx)
	if err != nil {
		return pgContainer, nil, err
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		return pgContainer, nil, err
	}
	dbname := "postgres"
	db, err = newPostgres(ip, port.Port(), "postgres", "password", dbname, SSLModeDisable)

	return pgContainer, db, err
}

func startCockroach() (container testcontainers.Container, dn *sql, err error) {
	ctx := context.Background()
	crOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "cockroachdb/cockroach:" + cockroachVersion,
			ExposedPorts: []string{"26257/tcp", "8080/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
			WaitingFor: wait.ForAll(
				wait.ForLog("initialized new cluster"),
				wait.ForListeningPort("8080/tcp"),
				wait.ForListeningPort("26257/tcp"),
			),
			Cmd: []string{"start-single-node", "--insecure", "--store=type=mem,size=70%"},
		}
		crContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := crContainer.Host(ctx)
	if err != nil {
		return crContainer, nil, err
	}
	port, err := crContainer.MappedPort(ctx, "26257")
	if err != nil {
		return crContainer, nil, err
	}
	dbname := "defaultdb"
	db, err := newPostgres(ip, port.Port(), "root", "password", dbname, SSLModeDisable)

	return crContainer, db, err
}

func startRedis() (container testcontainers.Container, s *redis, err error) {
	ctx := context.Background()
	redisOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "redis:" + redisVersion,
			ExposedPorts: []string{"6379/tcp"},
			Tmpfs:        map[string]string{"/data": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Ready to accept connections"),
				wait.ForListeningPort("6379/tcp"),
			),
		}
		redisContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := redisContainer.Host(ctx)
	if err != nil {
		return redisContainer, nil, err
	}
	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return redisContainer, nil, err
	}
	db, err := newRedis(ctx, ip, port.Port())
	if err != nil {
		return redisContainer, nil, err
	}
	return redisContainer, db, nil
}

func startEtcd() (container testcontainers.Container, s *etcd, err error) {
	ctx := context.Background()
	etcdOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "quay.io/coreos/etcd:" + etcdVersion,
			ExposedPorts: []string{"2379/tcp", "2380/tcp"},
			Cmd: []string{"etcd",
				"--name", "etcd",
				"--advertise-client-urls", "http://0.0.0.0:2379",
				"--initial-advertise-peer-urls", "http://0.0.0.0:2380",
				"--listen-client-urls", "http://0.0.0.0:2379",
				"--listen-peer-urls", "http://0.0.0.0:2380",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("ready to serve client requests"),
				wait.ForListeningPort("2379/tcp"),
				wait.ForListeningPort("2380/tcp"),
			),
		}
		etcdContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := etcdContainer.Host(ctx)
	if err != nil {
		return etcdContainer, nil, err
	}
	port, err := etcdContainer.MappedPort(ctx, "2379")
	if err != nil {
		return etcdContainer, nil, err
	}
	db, err := newEtcd(ctx, ip, port.Port(), nil, nil, true)
	if err != nil {
		return etcdContainer, nil, err
	}
	return etcdContainer, db, nil
}

func startMongodb() (container testcontainers.Container, s *mongodb, err error) {
	ctx := context.Background()

	mdbOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        `mongo:` + mdbVersion,
			ExposedPorts: []string{`27017/tcp`},
			Env: map[string]string{
				`MONGO_INITDB_ROOT_USERNAME`: `testuser`,
				`MONGO_INITDB_ROOT_PASSWORD`: `testuser`,
			},
			WaitingFor: wait.ForAll(
				wait.ForLog(`Waiting for connections`),
				wait.ForListeningPort(`27017/tcp`),
			),
			Cmd: []string{`mongod`},
		}
		mdbContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := mdbContainer.Host(ctx)
	if err != nil {
		return mdbContainer, nil, err
	}
	port, err := mdbContainer.MappedPort(ctx, `27017`)
	if err != nil {
		return mdbContainer, nil, err
	}

	opts := options.Client()
	opts.ApplyURI(fmt.Sprintf(`mongodb://%s:%s`, ip, port.Port()))
	opts.Auth = &options.Credential{
		AuthMechanism: `SCRAM-SHA-1`,
		Username:      `testuser`,
		Password:      `testuser`,
	}

	c := MongoConfig{
		DatabaseName:       `go-ipam`,
		MongoClientOptions: opts,
	}
	db, err := newMongo(ctx, c)

	return mdbContainer, db, err
}

func startKeyDB() (container testcontainers.Container, s *redis, err error) {
	ctx := context.Background()
	keyDBOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "eqalpha/keydb:" + keyDBVersion,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Server initialized"),
				wait.ForListeningPort("6379/tcp"),
			),
		}
		keyDBContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := keyDBContainer.Host(ctx)
	if err != nil {
		return keyDBContainer, nil, err
	}
	port, err := keyDBContainer.MappedPort(ctx, "6379")
	if err != nil {
		return keyDBContainer, nil, err
	}
	db, err := newRedis(ctx, ip, port.Port())
	if err != nil {
		return keyDBContainer, nil, err
	}
	return keyDBContainer, db, nil
}

// func stopDB(c testcontainers.Container) error {
// 	ctx := context.Background()
// 	return c.Terminate(ctx)
// }

// func cleanUp(s *sql) {
// 	s.db.MustExec("DROP TABLE prefixes")
// }

// cleanable interface for impls that support cleaning before each testrun
type cleanable interface {
	cleanup() error
}

// postCleanable interface for impls that support cleaning after each testrun
type postCleanable interface {
	postCleanup() error
}

// extendedSQL extended sql interface
type extendedSQL struct {
	*sql
	c testcontainers.Container
}

// extendedSQL extended sql interface
type kvStorage struct {
	*redis
	c testcontainers.Container
}
type kvEtcdStorage struct {
	*etcd
	c testcontainers.Container
}

type docStorage struct {
	*mongodb
	c testcontainers.Container
}

func newLocalFileWithCleanup() (*file, error) {
	ctx := context.Background()
	fp, err := os.CreateTemp("", "go-ipam-*.json")

	if err != nil {
		return nil, err
	}

	err = fp.Close()
	if err != nil {
		return nil, err
	}

	return &file{
		path:       fp.Name(),
		prettyJSON: true,
		parent:     NewMemory(ctx),
		modTime:    nullModTime,
		lock:       sync.RWMutex{},
	}, nil
}

func newPostgresWithCleanup() (*extendedSQL, error) {
	c, s, err := startPostgres()
	if err != nil {
		return nil, err
	}

	ext := &extendedSQL{
		sql: s,
		c:   c,
	}

	return ext, nil
}
func newCockroachWithCleanup() (*extendedSQL, error) {
	c, s, err := startCockroach()
	if err != nil {
		return nil, err
	}

	ext := &extendedSQL{
		sql: s,
		c:   c,
	}

	return ext, nil
}
func newRedisWithCleanup() (*kvStorage, error) {
	c, r, err := startRedis()
	if err != nil {
		return nil, err
	}

	kv := &kvStorage{
		redis: r,
		c:     c,
	}

	return kv, nil
}
func newEtcdWithCleanup() (*kvEtcdStorage, error) {
	c, r, err := startEtcd()
	if err != nil {
		return nil, err
	}

	kv := &kvEtcdStorage{
		etcd: r,
		c:    c,
	}

	return kv, nil
}

func newKeyDBWithCleanup() (*kvStorage, error) {
	c, r, err := startKeyDB()
	if err != nil {
		return nil, err
	}

	kv := &kvStorage{
		redis: r,
		c:     c,
	}

	return kv, nil
}

func newMongodbWithCleanup() (*docStorage, error) {
	c, s, err := startMongodb()
	if err != nil {
		return nil, err
	}

	x := &docStorage{
		mongodb: s,
		c:       c,
	}
	return x, nil
}

func (f *file) cleanup() error {
	if err := f.clearParent(context.Background()); err != nil {
		return err
	}
	if _, err := os.Stat(f.path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return os.Remove(f.path)
}

func (f *file) postCleanup() error {
	return f.cleanup()
}

// cleanup database before test
func (e *extendedSQL) cleanup() error {
	tx := e.db.MustBegin()
	_, err := e.db.Exec("TRUNCATE TABLE prefixes")
	if err != nil {
		return err
	}
	return tx.Commit()
}

// cleanup database before test
func (kv *kvStorage) cleanup() error {
	return kv.DeleteAllPrefixes(context.Background(), defaultNamespace)
}

// cleanup database before test
func (kv *kvEtcdStorage) cleanup() error {
	return kv.DeleteAllPrefixes(context.Background(), defaultNamespace)
}

// cleanup database before test
func (sql *sql) cleanup() error {
	tx := sql.db.MustBegin()
	_, err := sql.db.Exec("TRUNCATE TABLE prefixes")
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (ds *docStorage) cleanup() error {
	return ds.DeleteAllPrefixes(context.Background(), defaultNamespace)
}

type benchMethod func(b *testing.B, ipam *ipamer)

func benchWithBackends(b *testing.B, fn benchMethod) {
	for _, storageProvider := range storageProviders() {
		if backend != "" && backend != storageProvider.name {
			continue
		}
		storage := storageProvider.provide()

		if tp, ok := storage.(cleanable); ok {
			err := tp.cleanup()
			if err != nil {
				b.Errorf("error cleaning up before the test: %v", err)
			}
		}

		ipamer := &ipamer{storage: storage}
		testName := storageProvider.name

		b.Run(testName, func(b *testing.B) {
			fn(b, ipamer)
		})

		if tp, ok := storage.(postCleanable); ok {
			err := tp.postCleanup()
			if err != nil {
				b.Errorf("error cleaning up after the test: %v", err)
			}
		}
	}
}

type testMethod func(t *testing.T, ipam *ipamer)

func testWithBackends(t *testing.T, fn testMethod) {
	t.Helper()
	// prevent testcontainer logging mangle test and benchmark output
	for _, storageProvider := range storageProviders() {
		if backend != "" && backend != storageProvider.name {
			continue
		}
		storage := storageProvider.provide()

		if tp, ok := storage.(cleanable); ok {
			err := tp.cleanup()
			if err != nil {
				t.Errorf("error cleaning up, %v", err)
			}
		}

		ipamer := &ipamer{storage: storage}
		testName := storageProvider.name

		t.Run(testName, func(t *testing.T) {
			fn(t, ipamer)
		})

		if tp, ok := storage.(postCleanable); ok {
			err := tp.postCleanup()
			if err != nil {
				t.Errorf("error cleaning up after the test: %v", err)
			}
		}
	}
}

type sqlTestMethod func(t *testing.T, sql *sql)

func testWithSQLBackends(t *testing.T, fn sqlTestMethod) {
	t.Helper()
	// prevent testcontainer logging mangle test and benchmark output
	for _, storageProvider := range storageProviders() {
		if backend != "" && backend != storageProvider.name {
			continue
		}
		sqlstorage := storageProvider.providesql()
		if sqlstorage == nil {
			continue
		}

		err := sqlstorage.cleanup()
		if err != nil {
			t.Errorf("error cleaning up, %v", err)
		}

		testName := storageProvider.name

		t.Run(testName, func(t *testing.T) {
			fn(t, sqlstorage)
		})
	}
}

type provide func() Storage
type providesql func() *sql

// storageProvider provides different storages
type storageProvider struct {
	name       string
	provide    provide
	providesql providesql
}

func storageProviders() []storageProvider {
	return []storageProvider{
		{
			name: "Memory",
			provide: func() Storage {
				return NewMemory(context.Background())
			},
			providesql: func() *sql {
				return nil
			},
		},
		{
			name: "File",
			provide: func() Storage {
				storage, err := newLocalFileWithCleanup()
				if err != nil {
					panic("error getting local file storage")
				}
				return storage
			},
			providesql: func() *sql {
				return nil
			},
		},
		{
			name: "Postgres",
			provide: func() Storage {
				storage, err := newPostgresWithCleanup()
				if err != nil {
					panic("error getting postgres storage")
				}
				return storage
			},
			providesql: func() *sql {
				storage, err := newPostgresWithCleanup()
				if err != nil {
					panic("error getting postgres storage")
				}
				return storage.sql
			},
		},
		{
			name: "Cockroach",
			provide: func() Storage {
				storage, err := newCockroachWithCleanup()
				if err != nil {
					panic("error getting cockroach storage")
				}
				return storage
			},
			providesql: func() *sql {
				storage, err := newCockroachWithCleanup()
				if err != nil {
					panic("error getting cockroach storage")
				}
				return storage.sql
			},
		},
		{
			name: "Redis",
			provide: func() Storage {
				s, err := newRedisWithCleanup()
				if err != nil {
					panic(fmt.Sprintf("unable to start redis:%s", err))
				}
				return s
			},
			providesql: func() *sql {
				return nil
			},
		},
		{
			name: "Etcd",
			provide: func() Storage {
				s, err := newEtcdWithCleanup()
				if err != nil {
					panic(fmt.Sprintf("unable to start etcd:%s", err))
				}
				return s
			},
			providesql: func() *sql {
				return nil
			},
		},
		{
			name: "KeyDB",
			provide: func() Storage {
				s, err := newKeyDBWithCleanup()
				if err != nil {
					panic(fmt.Sprintf("unable to start keydb:%s", err))
				}
				return s
			},
			providesql: func() *sql {
				return nil
			},
		},
		{
			name: "MongoDB",
			provide: func() Storage {
				storage, err := newMongodbWithCleanup()
				if err != nil {
					panic(fmt.Sprintf(`error getting mongodb storage, error: %s`, err))
				}
				return storage
			},
			providesql: func() *sql {
				return nil
			},
		},
	}
}
