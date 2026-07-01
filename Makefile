.PHONY: run test test-integration build db-up db-down db-psql

run:
	go run ./cmd/server

test:
	go test ./...

test-integration:
	go test ./tests/integration/... -count=1

build:
	go build -o bin/server ./cmd/server

# --- Local development database (PostgreSQL via Docker) ---

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-psql:
	docker compose exec postgres psql -U ahrm -d ahrm
