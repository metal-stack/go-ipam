// Package backends wires all supported Storage backends into the shared
// testregistry. It is intended to be imported (blank) by tests only, so that
// the core ipam package and its consumers never transitively depend on any
// database driver.
package backends

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo/options"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/etcd"
	"github.com/metal-stack/go-ipam/pkg/file"
	"github.com/metal-stack/go-ipam/pkg/mongodb"
	"github.com/metal-stack/go-ipam/pkg/postgres"
	"github.com/metal-stack/go-ipam/pkg/redis"
	"github.com/metal-stack/go-ipam/pkg/testregistry"
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
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func init() {
	pgVersion = envOr("PG_VERSION", "18-alpine")
	cockroachVersion = envOr("COCKROACH_VERSION", "latest-v24.3")
	redisVersion = envOr("REDIS_VERSION", "8.6-alpine")
	keyDBVersion = envOr("KEYDB_VERSION", "latest")
	etcdVersion = envOr("ETCD_VERSION", "v3.7.0")
	mdbVersion = envOr("MONGODB_VERSION", "7")

	testregistry.Register(testregistry.Provider{
		Name: "Memory",
		Provide: func(ctx context.Context) (any, error) {
			return ipam.NewMemory(ctx), nil
		},
	})
	testregistry.Register(testregistry.Provider{
		Name: "File",
		Provide: func(ctx context.Context) (any, error) {
			fp, err := os.CreateTemp("", "go-ipam-*.json")
			if err != nil {
				return nil, err
			}
			if err := fp.Close(); err != nil {
				return nil, err
			}
			return &fileStorage{Storage: file.NewLocalFile(ctx, fp.Name()), path: fp.Name()}, nil
		},
		PostCleanup: func(_ context.Context, storage any) error {
			fs, ok := storage.(*fileStorage)
			if !ok {
				return nil
			}
			if _, err := os.Stat(fs.path); err != nil {
				return nil
			}
			return os.Remove(fs.path)
		},
	})
	testregistry.Register(testregistry.Provider{
		Name: "Postgres",
		Provide: func(ctx context.Context) (any, error) {
			return startPostgres(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
	testregistry.Register(testregistry.Provider{
		Name: "Cockroach",
		Provide: func(ctx context.Context) (any, error) {
			return startCockroach(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
	testregistry.Register(testregistry.Provider{
		Name: "Redis",
		Provide: func(ctx context.Context) (any, error) {
			return startRedis(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
	testregistry.Register(testregistry.Provider{
		Name: "KeyDB",
		Provide: func(ctx context.Context) (any, error) {
			return startKeyDB(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
	testregistry.Register(testregistry.Provider{
		Name: "Etcd",
		Provide: func(ctx context.Context) (any, error) {
			return startEtcd(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
	testregistry.Register(testregistry.Provider{
		Name: "MongoDB",
		Provide: func(ctx context.Context) (any, error) {
			return startMongodb(ctx)
		},
		Cleanup:     cleanupNamespaces,
		PostCleanup: cleanupNamespaces,
	})
}

type fileStorage struct {
	ipam.Storage
	path string
}

// cleanupNamespaces removes all prefixes and non-default namespaces so that
// every test/benchmark starts from a clean state.
func cleanupNamespaces(ctx context.Context, storage any) error {
	s, ok := storage.(ipam.Storage)
	if !ok {
		return fmt.Errorf("storage is not a ipam.Storage: %T", storage)
	}
	namespaces, err := s.ListNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		if err := s.DeleteAllPrefixes(ctx, namespace); err != nil {
			return err
		}
		if namespace == ipam.DefaultNamespace {
			continue
		}
		if err := s.DeleteNamespace(ctx, namespace); err != nil {
			return err
		}
	}
	return nil
}

func startPostgres(ctx context.Context) (ipam.Storage, error) {
	pgOnce.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "postgres:" + pgVersion,
			ExposedPorts: []string{"5432/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
			Tmpfs:        map[string]string{"/var/lib/postgresql": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
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
		return nil, err
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}
	return postgres.New(ip, port.Port(), "postgres", "password", "postgres", postgres.SSLModeDisable)
}

func startCockroach(ctx context.Context) (ipam.Storage, error) {
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
		return nil, err
	}
	port, err := crContainer.MappedPort(ctx, "26257")
	if err != nil {
		return nil, err
	}
	return postgres.New(ip, port.Port(), "root", "password", "defaultdb", postgres.SSLModeDisable)
}

func startRedis(ctx context.Context) (ipam.Storage, error) {
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
		return nil, err
	}
	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}
	return redis.NewRedis(ctx, ip, port.Port())
}

func startKeyDB(ctx context.Context) (ipam.Storage, error) {
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
		return nil, err
	}
	port, err := keyDBContainer.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}
	return redis.NewRedis(ctx, ip, port.Port())
}

func startEtcd(ctx context.Context) (ipam.Storage, error) {
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
		return nil, err
	}
	port, err := etcdContainer.MappedPort(ctx, "2379")
	if err != nil {
		return nil, err
	}
	return etcd.NewEtcd(ctx, ip, port.Port(), nil, nil, true)
}

func startMongodb(ctx context.Context) (ipam.Storage, error) {
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
		return nil, err
	}
	port, err := mdbContainer.MappedPort(ctx, `27017`)
	if err != nil {
		return nil, err
	}

	opts := options.Client()
	opts.ApplyURI(fmt.Sprintf(`mongodb://%s:%s`, ip, port.Port()))
	opts.Auth = &options.Credential{
		AuthMechanism: `SCRAM-SHA-1`,
		Username:      `testuser`,
		Password:      `testuser`,
	}

	c := mongodb.MongoConfig{
		DatabaseName:       `go-ipam`,
		MongoClientOptions: opts,
	}
	return mongodb.NewMongo(ctx, c)
}
