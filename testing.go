package ipam

import (
	"context"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	once      sync.Once
	postgresC testcontainers.Container
)

func startPostgres() (container testcontainers.Container, dn *sql, err error) {
	ctx := context.Background()
	once.Do(func() {
		var err error
		req := testcontainers.ContainerRequest{
			Image:        "postgres:12-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env:          map[string]string{"POSTGRES_PASSWORD": "password"},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			),
			Cmd: []string{"postgres", "-c", "max_connections=200"},
		}
		postgresC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := postgresC.Host(ctx)
	if err != nil {
		return postgresC, nil, err
	}
	port, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		return postgresC, nil, err
	}
	dbname := "postgres"
	db, err := NewPostgresStorage(ip, port.Port(), "postgres", "password", dbname, SSLModeDisable)

	return postgresC, db, err
}

// func stopDB(c testcontainers.Container) error {
// 	ctx := context.Background()
// 	return c.Terminate(ctx)
// }

func cleanUp(s *sql) {
	s.db.MustExec("DROP TABLE prefixes")
}

// Cleanable interface for impls that support cleaning before each testrun
type Cleanable interface {
	cleanup() error
}

// ExtendedSQL extended sql interface
type ExtendedSQL struct {
	*sql
	c testcontainers.Container
}

func newPostgresWithCleanup() (*ExtendedSQL, error) {
	c, s, err := startPostgres()
	if err != nil {
		return nil, err
	}

	ext := &ExtendedSQL{
		sql: s,
		c:   c,
	}

	return ext, nil
}

// cleanup database before test
func (e *ExtendedSQL) cleanup() error {
	tx := e.sql.db.MustBegin()
	_, err := e.sql.db.Exec("TRUNCATE TABLE prefixes")
	if err != nil {
		return err
	}
	return tx.Commit()
}

type testMethod func(t *testing.T, ipam *ipamer)

func testWithBackends(t *testing.T, fn testMethod) {
	for _, storageProvider := range storageProviders() {

		storage := storageProvider.provide()

		if tp, ok := storage.(Cleanable); ok {
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
	}
}

type provide func() Storage

// StorageProvider provides differen storages
type StorageProvider struct {
	name    string
	provide provide
}

func storageProviders() []StorageProvider {
	return []StorageProvider{
		{
			name: "Memory",
			provide: func() Storage {
				return NewMemory()
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
		},
	}
}
