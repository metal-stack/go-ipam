module github.com/metal-stack/go-ipam

go 1.13

require (
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.3.0
	// sqlite v2.x is a unfortunate release
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	github.com/testcontainers/testcontainers-go v0.5.0
)
