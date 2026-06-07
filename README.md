# AHRM Options Arbitrage Scanner

Go monolith for AHRM options arbitrage scanning and market indicators.

## Phase 1: Project Skeleton

Phase 1 provides configuration loading, logging, health/readiness endpoints, optional Supabase PostgreSQL connectivity, and a minimal SQL migration runner (`schema_migrations` only).

## Requirements

- Go 1.22+
- Optional: Supabase PostgreSQL credentials in a local `.env` file

## Setup

```bash
cp .env.example .env
# Edit .env when you are ready to connect Supabase
go mod tidy
```

## Run

```bash
make run
# or
go run ./cmd/server
```

## Test

```bash
make test
```

Integration DB test (skipped without Supabase env):

```bash
make test-integration
```

## Endpoints

- `GET /health` — always returns `{"status":"ok"}`
- `GET /ready` — readiness report; returns `503` if DB pool exists but ping fails

## Manual checks

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/ready
```
