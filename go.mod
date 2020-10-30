module github.com/metal-stack/go-ipam

go 1.15

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	// sqlite v2.x is a unfortunate release
	github.com/mattn/go-sqlite3 v1.14.4 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/testcontainers/testcontainers-go v0.9.0
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
	golang.org/x/net v0.0.0-20201029221708-28c70e62bb1d // indirect
)
