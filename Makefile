.PHONY: run test test-integration build

run:
	go run ./cmd/server

test:
	go test ./...

test-integration:
	go test ./tests/integration/... -count=1

build:
	go build -o bin/server ./cmd/server
