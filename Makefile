.ONESHELL:
CGO_ENABLED := $(or ${CGO_ENABLED},0)
GO := go
GO111MODULE := on
PG_VERSION := $(or ${PG_VERSION},"13-alpine")
COCKROACH_VERSION := $(or ${COCKROACH_VERSION},"v20.2.7")

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

all: test bench

.PHONY: bench
bench:
	CGO_ENABLED=1 $(GO) test -bench . -run=- -count 5 -benchmem

.PHONY: benchstat
benchstat:
	git stash
	CGO_ENABLED=1 $(GO) test -bench . -run=- -count 5 -benchmem > old.txt
	git stash pop
	CGO_ENABLED=1 $(GO) test -bench . -run=- -count 5 -benchmem > new.txt
	benchstat old.txt new.txt

.PHONY: test
test:
	CGO_ENABLED=1 $(GO) test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out

.PHONY: golangcicheck
golangcicheck:
	@/bin/bash -c "type -P golangci-lint;" 2>/dev/null || (echo "golangci-lint is required but not available in current PATH. Install: https://github.com/golangci/golangci-lint#install"; exit 1)

.PHONY: lint
lint: golangcicheck
	golangci-lint run

.PHONY: cockroach-up
cockroach-up: cockroach-rm postgres-rm
	# https://www.cockroachlabs.com/docs/v19.2/start-a-local-cluster-in-docker-linux.html#main-content
	docker network create -d bridge roachnet
	docker run -d --name=roach1 --hostname=roach1 --net=roachnet -p 5433:26257 -p 8080:8080 cockroachdb/cockroach:$(COCKROACH_VERSION) start-single-node --insecure --listen-addr=0.0.0.0

.PHONY: cockroach-up-cluster
cockroach-up-cluster: cockroach-rm
	# https://www.cockroachlabs.com/docs/v19.2/start-a-local-cluster-in-docker-linux.html#main-content
	docker network create -d bridge roachnet
	docker run -d --name=roach1 --hostname=roach1 --net=roachnet -p 5433:26257 -p 8080:8080 cockroachdb/cockroach:$(COCKROACH_VERSION) start --insecure --join=roach1,roach2,roach3
	docker run -d --name=roach2 --hostname=roach2 --net=roachnet cockroachdb/cockroach:$(COCKROACH_VERSION) start --insecure --join=roach1,roach2,roach3
	docker run -d --name=roach3 --hostname=roach3 --net=roachnet cockroachdb/cockroach:$(COCKROACH_VERSION) start --insecure --join=roach1,roach2,roach3
	docker exec -it roach1 ./cockroach init --insecure

.PHONY: cockroach-rm
cockroach-rm:
	docker rm -f roach1 roach2 roach3 || true
	docker network rm roachnet || true

.PHONY: protoc
protoc:
	docker run --rm --user $$(id -u):$$(id -g) -v ${PWD}:/work metalstack/builder protoc --proto_path=server/api --go_out=plugins=grpc:server/api server/api/v1/*.proto
	docker run --rm --user $$(id -u):$$(id -g) -v ${PWD}:/work metalstack/builder protoc --proto_path=server/api --go_out=plugins=grpc:server/api server/api/grpc/health/v1/*.proto

.PHONY: server
server:
	go build -tags netgo -ldflags "-X 'github.com/metal-stack/v.Version=$(VERSION)' \
								   -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
								   -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
								   -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" \
						 -o bin/server server/main.go
	strip bin/server

.PHONY: client
client:
	go build -tags netgo -o bin/client server/cli/main.go
	strip bin/client

.PHONY: postgres-up
postgres-up: postgres-rm
	docker run -d --name ipamdb -p 5433:5432 -e POSTGRES_PASSWORD="password" -e POSTGRES_USER="ipam" -e POSTGRES_DB="ipam" postgres:13-alpine

.PHONY: postgres-rm
postgres-rm:
	docker rm -f ipamdb || true

.PHONY: certs
certs:
	cd certs && cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
	cd certs && cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile client-server server.json | cfssljson -bare server -
	cd certs && cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile client client.json | cfssljson -bare client -
