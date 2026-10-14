package mongodb

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
	"github.com/metal-stack/go-ipam/pkg/test/suite"
)

var (
	mdbOnce      sync.Once
	mdbContainer testcontainers.Container
	mdbVersion   string
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func startMongodb(tb testing.TB) *mongodb {
	mdbOnce.Do(func() {
		mdbVersion = envOr("MONGODB_VERSION", "7")
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
		tb.Fatal(err)
	}
	port, err := mdbContainer.MappedPort(ctx, `27017`)
	if err != nil {
		tb.Fatal(err)
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
	var lastErr error
	for range 30 {
		m, err := newMongo(ctx, c)
		if err == nil {
			return m
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	tb.Fatal(lastErr)
	return nil
}

func provide(tb testing.TB) ipam.Storage {
	s := startMongodb(tb)
	ctx := tb.Context()
	if err := suite.Cleanup(ctx, s); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = suite.Cleanup(context.Background(), s)
	})
	return s
}

func TestConformance(t *testing.T) {
	suite.Run(t, provide)
}
