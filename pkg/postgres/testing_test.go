package postgres

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgOnce           sync.Once
	pgContainer      testcontainers.Container
	pgVersion        string
	crOnce           sync.Once
	crContainer      testcontainers.Container
	cockroachVersion string
	backend          string
)

func TestMain(m *testing.M) {
	pgVersion = os.Getenv("PG_VERSION")
	if pgVersion == "" {
		pgVersion = "18-alpine"
	}
	cockroachVersion = os.Getenv("COCKROACH_VERSION")
	if cockroachVersion == "" {
		cockroachVersion = "latest-v24.3"
	}
	backend = os.Getenv("BACKEND")
	os.Exit(m.Run())
}

func startPostgres(ctx context.Context) (container testcontainers.Container, db *sql, err error) {
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

func startCockroach(ctx context.Context) (container testcontainers.Container, dn *sql, err error) {
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

type sqlTestMethod func(t *testing.T, sql *sql)

func testWithSQLBackends(t *testing.T, fn sqlTestMethod) {
	t.Helper()
	for _, provider := range []struct {
		name    string
		provide func(ctx context.Context) (*sql, error)
	}{
		{name: "Postgres", provide: func(ctx context.Context) (*sql, error) {
			_, s, err := startPostgres(ctx)
			return s, err
		}},
		{name: "Cockroach", provide: func(ctx context.Context) (*sql, error) {
			_, s, err := startCockroach(ctx)
			return s, err
		}},
	} {
		if backend != "" && backend != provider.name {
			continue
		}
		sqlstorage, err := provider.provide(t.Context())
		if err != nil {
			t.Fatalf("error getting %s storage: %v", provider.name, err)
		}
		if err := sqlstorage.cleanup(); err != nil {
			t.Errorf("error cleaning up, %v", err)
		}
		t.Run(provider.name, func(t *testing.T) {
			fn(t, sqlstorage)
		})
	}
}

func (sql *sql) cleanup() error {
	tx := sql.db.MustBegin()
	_, err := sql.db.Exec("TRUNCATE TABLE prefixes")
	if err != nil {
		return err
	}
	return tx.Commit()
}
