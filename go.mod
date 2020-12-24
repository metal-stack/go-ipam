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
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	inet.af/netaddr v0.0.0-20201223185330-97d366981fac
)
