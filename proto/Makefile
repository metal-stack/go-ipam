.PHONY: protolint
protolint:
	buf format -w api/v1
	buf lint -v

.PHONY: protoc
protoc: protolint
	buf generate -v