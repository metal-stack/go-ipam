package ipam

import (
	"context"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	once      sync.Once
	postgresC testcontainers.Container
)

func startDB() (container testcontainers.Container, dn *sql, err error) {
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
	db, err := NewPostgresStorage(ip, port.Port(), "postgres", "password", dbname, "disable")

	return postgresC, db, err
}

// func stopDB(c testcontainers.Container) error {
// 	ctx := context.Background()
// 	return c.Terminate(ctx)
// }

func cleanUp(s *sql) {
	s.db.MustExec("DROP TABLE prefixes")
}
