package ipam

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	// import for sqlx to use postgres driver
	_ "github.com/lib/pq"
)

const postgresSchema = `
CREATE TABLE IF NOT EXISTS prefixes (
	cidr   text PRIMARY KEY NOT NULL,
	prefix JSONB
);

CREATE INDEX IF NOT EXISTS prefix_idx ON prefixes USING GIN(prefix);
`

// NewPostgresTransactionalStorage creates a new Storage which uses postgres.
func NewPostgresTransactionalStorage(host, port, user, password, dbname, sslmode string) (*tsql, error) {
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", host, port, user, dbname, password, sslmode))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database:%v", err)
	}
	db.MustExec(postgresSchema)
	return &tsql{
		db: db,
	}, nil
}

// NewPostgresStorage creates a new Storage which uses postgres.
func NewPostgresStorage(host, port, user, password, dbname, sslmode string) (*sql, error) {
	t, err := NewPostgresTransactionalStorage(host, port, user, password, dbname, sslmode)
	if err != nil {
		return nil, err
	}
	return &sql{
		tsql: t,
	}, nil
}
