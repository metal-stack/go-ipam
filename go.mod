module github.com/metal-stack/go-ipam

go 1.21

require (
	connectrpc.com/connect v1.14.0
	connectrpc.com/grpchealth v1.3.0
	connectrpc.com/grpcreflect v1.2.0
	connectrpc.com/otelconnect v0.6.0
	github.com/avast/retry-go/v4 v4.5.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/klauspost/connect-compress/v2 v2.0.0
	github.com/lib/pq v1.10.9
	github.com/metal-stack/v v1.0.3
	github.com/prometheus/client_golang v1.18.0
	github.com/redis/go-redis/v9 v9.4.0
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.27.0
	github.com/urfave/cli/v2 v2.27.1
	go.etcd.io/etcd/client/v3 v3.5.11
	go.mongodb.org/mongo-driver v1.13.1
	go.opentelemetry.io/otel/exporters/prometheus v0.44.0
	go.opentelemetry.io/otel/sdk/metric v1.21.0
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba
	golang.org/x/net v0.23.0
	golang.org/x/sync v0.6.0
	google.golang.org/protobuf v1.32.0
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.11.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/containerd v1.7.11 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/lufia/plan9stats v0.0.0-20231016141302-07b5767bb0ed // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc5 // indirect
	github.com/opencontainers/runc v1.1.11 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20221212215047-62379fc7944b // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil/v3 v3.23.12 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xrash/smetrics v0.0.0-20231213231151-1d8dd44e695e // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	go.etcd.io/etcd/api/v3 v3.5.11 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.11 // indirect
	go.opentelemetry.io/otel v1.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.21.0 // indirect
	go.opentelemetry.io/otel/sdk v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.21.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/genproto v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/grpc v1.60.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
