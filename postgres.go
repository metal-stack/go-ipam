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

// NewPostgresStorage creates a new Storage which uses postgres.
func NewPostgresStorage(host, port, user, password, dbname string, ssl bool) (*sql, error) {
	db, err := sqlx.Connect("postgres", dataSource(host, port, user, password, dbname, ssl))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database:%v", err)
	}
	db.MustExec(postgresSchema)
	return &sql{
		db: db,
	}, nil
}

func dataSource(host, port, user, password, dbname string, ssl bool) string {
	sslmode := "sslmode=disable"
	if ssl {
		sslmode = "sslmode=enable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?%s", user, password, host, port, dbname, sslmode)
}
