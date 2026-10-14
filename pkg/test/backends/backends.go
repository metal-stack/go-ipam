// Package backends wires every supported Storage backend into the shared
// benchmark suite providers. It is intended for tests/benchmarks only so the
// core ipam package and its consumers never transitively depend on a database
// driver.
package backends

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo/options"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/etcd"
	"github.com/metal-stack/go-ipam/pkg/file"
	"github.com/metal-stack/go-ipam/pkg/mongodb"
	"github.com/metal-stack/go-ipam/pkg/postgres"
	"github.com/metal-stack/go-ipam/pkg/redis"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
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
}

// Providers returns all supported storage backends as benchmark providers.
func Providers() []suite.BenchProvider {
	return []suite.BenchProvider{
		{Name: "Memory", Provide: func(tb testing.TB) ipam.Storage {
			return ipam.NewMemory(tb.Context())
		}},
		{Name: "File", Provide: func(tb testing.TB) ipam.Storage {
			fp, err := os.CreateTemp("", "go-ipam-*.json")
			if err != nil {
				tb.Fatal(err)
			}
			if err := fp.Close(); err != nil {
				tb.Fatal(err)
			}
			tb.Cleanup(func() {
				_ = os.Remove(fp.Name())
			})
			return file.New(tb.Context(), fp.Name())
		}},
		{Name: "Postgres", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startPostgres(tb)
		})},
		{Name: "Cockroach", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startCockroach(tb)
		})},
		{Name: "Redis", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startRedis(tb)
		})},
		{Name: "KeyDB", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startKeyDB(tb)
		})},
		{Name: "Etcd", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startEtcd(tb)
		})},
		{Name: "MongoDB", Provide: cleaning(func(tb testing.TB) (ipam.Storage, error) {
			return startMongodb(tb)
		})},
	}
}

// cleaning wraps a backend constructor so that each benchmark starts from a
// clean storage and cleans up afterwards.
func cleaning(create func(tb testing.TB) (ipam.Storage, error)) suite.Provider {
	return func(tb testing.TB) ipam.Storage {
		s, err := create(tb)
		if err != nil {
			tb.Fatal(err)
		}
		ctx := tb.Context()
		if err := suite.Cleanup(ctx, s); err != nil {
			tb.Fatal(err)
		}
		tb.Cleanup(func() {
			_ = suite.Cleanup(context.Background(), s)
		})
		return s
	}
}

func startPostgres(tb testing.TB) (ipam.Storage, error) {
	pgOnce.Do(func() {
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
		var err error
		pgContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
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

func startCockroach(tb testing.TB) (ipam.Storage, error) {
	crOnce.Do(func() {
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
		var err error
		crContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
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

func startRedis(tb testing.TB) (ipam.Storage, error) {
	redisOnce.Do(func() {
		req := testcontainers.ContainerRequest{
			Image:        "redis:" + redisVersion,
			ExposedPorts: []string{"6379/tcp"},
			Tmpfs:        map[string]string{"/data": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Ready to accept connections"),
				wait.ForListeningPort("6379/tcp"),
			),
		}
		var err error
		redisContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
	ip, err := redisContainer.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}
	return redis.New(ctx, ip, port.Port())
}

func startKeyDB(tb testing.TB) (ipam.Storage, error) {
	keyDBOnce.Do(func() {
		req := testcontainers.ContainerRequest{
			Image:        "eqalpha/keydb:" + keyDBVersion,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Server initialized"),
				wait.ForListeningPort("6379/tcp"),
			),
		}
		var err error
		keyDBContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
	ip, err := keyDBContainer.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := keyDBContainer.MappedPort(ctx, "6379")
	if err != nil {
		return nil, err
	}
	return redis.New(ctx, ip, port.Port())
}

func startEtcd(tb testing.TB) (ipam.Storage, error) {
	etcdOnce.Do(func() {
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
		var err error
		etcdContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
	ip, err := etcdContainer.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := etcdContainer.MappedPort(ctx, "2379")
	if err != nil {
		return nil, err
	}
	return etcd.New(ctx, ip, port.Port(), nil, nil, true)
}

func startMongodb(tb testing.TB) (ipam.Storage, error) {
	mdbOnce.Do(func() {
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
		var err error
		mdbContainer, err = testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ctx := tb.Context()
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

	var lastErr error
	for range 30 {
		m, err := mongodb.New(ctx, mongodb.MongoConfig{
			DatabaseName:       `go-ipam`,
			MongoClientOptions: opts,
		})
		if err == nil {
			return m, nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return nil, lastErr
}
