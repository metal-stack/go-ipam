.ONESHELL:
CGO_ENABLED := $(or ${CGO_ENABLED},0)
GO := go
GO111MODULE := on

all: test bench

.PHONY: bench
bench:
	CGO_ENABLED=1 $(GO) test -bench . -benchmem

.PHONY: test
test:
	CGO_ENABLED=1 $(GO) test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out

.PHONY: golangcicheck
golangcicheck:
	@/bin/bash -c "type -P golangci-lint;" 2>/dev/null || (echo "golangci-lint is required but not available in current PATH. Install: https://github.com/golangci/golangci-lint#install"; exit 1)

.PHONY: lint
lint: golangcicheck
	golangci-lint run

.PHONY: postgres-up
postgres-up: postgres-rm
	docker run -d --name ipamdb -p 5433:5432 -e POSTGRES_PASSWORD="password" postgres:12-alpine postgres -c 'max_connections=200'

.PHONY: postgres-rm
postgres-rm:
	docker rm -f ipamdb || true

