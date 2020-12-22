module github.com/metal-stack/go-ipam

go 1.15

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.9.0
	// sqlite v2.x is a unfortunate release
	github.com/mattn/go-sqlite3 v1.14.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/testcontainers/testcontainers-go v0.9.0
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620 // indirect
	golang.org/x/net v0.0.0-20201216054612-986b41b23924 // indirect
	inet.af/netaddr v0.0.0-20201218162718-658fec415e52
)
