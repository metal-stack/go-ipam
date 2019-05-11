.ONESHELL:
CGO_ENABLED := $(or ${CGO_ENABLED},0)                                                                                                                                                                              
GO := go                                                                                                                                                                                                           
GO111MODULE := on

.PHONY: test
test:
	CGO_ENABLED=1 $(GO) test -cover ./...

.PHONY: bench
bench:
	CGO_ENABLED=1 $(GO) test -bench . -benchmem

.PHONY: test-ci
test-ci:
	CGO_ENABLED=1 $(GO) test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out

.PHONY: example
example:
	$(GO) run example/main.go
