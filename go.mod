module github.com/metal-stack/go-ipam

go 1.15

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/gogo/status v1.1.0
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.9.0
	// sqlite v2.x is a unfortunate release
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/metal-stack/metal-lib v0.6.7
	github.com/metal-stack/security v0.4.0
	github.com/metal-stack/v v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.9.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	google.golang.org/grpc v1.35.0
	google.golang.org/grpc/examples v0.0.0-20210116000752-504caa93c539 // indirect
	google.golang.org/protobuf v1.25.0
	inet.af/netaddr v0.0.0-20210115183222-bffc12a571f6
)
