.ONESHELL:
CGO_ENABLED := $(or ${CGO_ENABLED},0)                                                                                                                                                                              
GO := go                                                                                                                                                                                                           
GO111MODULE := on

.PHONY: bench
bench:
	CGO_ENABLED=1 $(GO) test -bench . -benchmem

.PHONY: test
test:
	CGO_ENABLED=1 $(GO) test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out

.PHONY: postgres-up
postgres-up: postgres-rm
	docker run -d --name ipamdb -p 5433:5432 -e POSTGRES_PASSWORD="password" postgres

.PHONY: postgres-rm
postgres-rm:
	docker rm -f ipamdb || true

