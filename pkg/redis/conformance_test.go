package redis

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	ipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

var (
	redisOnce      sync.Once
	redisContainer testcontainers.Container
	redisVersion   string

	keyDBOnce      sync.Once
	keyDBContainer testcontainers.Container
	keyDBVersion   string
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func startRedis(tb testing.TB) *redis {
	redisOnce.Do(func() {
		redisVersion = envOr("REDIS_VERSION", "8.6-alpine")
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
		tb.Fatal(err)
	}
	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		tb.Fatal(err)
	}
	r, err := newRedis(ctx, ip, port.Port())
	if err != nil {
		tb.Fatal(err)
	}
	return r
}

func startKeyDB(tb testing.TB) *redis {
	keyDBOnce.Do(func() {
		keyDBVersion = envOr("KEYDB_VERSION", "latest")
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
		tb.Fatal(err)
	}
	port, err := keyDBContainer.MappedPort(ctx, "6379")
	if err != nil {
		tb.Fatal(err)
	}
	r, err := newRedis(ctx, ip, port.Port())
	if err != nil {
		tb.Fatal(err)
	}
	return r
}

func cleanProvider(start func(testing.TB) *redis) func(testing.TB) ipam.Storage {
	return func(tb testing.TB) ipam.Storage {
		s := start(tb)
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

func TestConformance(t *testing.T) {
	t.Run("Redis", func(t *testing.T) {
		suite.Run(t, cleanProvider(startRedis))
	})
	t.Run("KeyDB", func(t *testing.T) {
		suite.Run(t, cleanProvider(startKeyDB))
	})
}
