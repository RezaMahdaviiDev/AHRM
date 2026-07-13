# AHRM Options Arbitrage Scanner

Go monolith for AHRM (`اهرم`) options arbitrage, market indicators, dashboard, and Telegram alerts.

## Requirements

- Go 1.22+
- `.env` file (see `.env.example`)

## Setup

```bash
cp .env.example .env
go mod tidy
```

## Run

```bash
make run
```

## Local development database (PostgreSQL)

Production uses PostgreSQL (Supabase). For local development you can run the same engine
in Docker so there are no code or migration differences between environments — only the
`SUPABASE_*` connection values change.

```bash
make db-up                 # start PostgreSQL (Docker) in the background
cp .env.dev.example .env   # point the app at the local DB (SSL disabled)
make run                   # migrations run automatically on startup
make db-psql               # optional: open a psql shell
make db-down               # stop the DB (data is kept in the named volume)
```

When the DB is configured, `GET /ready` reports `"supabase":{"configured":true,"connected":true}`
and the breadth/advance-decline 10-day window is stored in the `market_daily_stats` table
instead of the JSON file fallback. Data persists in the `ahrm_pgdata` Docker volume across
restarts; remove it with `docker compose down -v`.

Integration tests run against this DB:

```bash
set -a && . ./.env && set +a   # export SUPABASE_* for the test process
make test-integration
```

Without `SUPABASE_*` set, integration tests self-skip and the app falls back to the JSON
file store (`data/market_history.json`).

## Test

```bash
make test
make test-integration   # requires Supabase .env
```

## Endpoints

| Path | Description |
|------|-------------|
| `GET /health` | Liveness (no external deps) |
| `GET /ready` | Readiness + Supabase status |
| `GET /dashboard` | Main dashboard |
| `GET /arbitrage` | Arbitrage opportunities |
| `GET /hv` | Historical volatility |
| `GET /market` | Breadth & Advance/Decline |
| `GET /matrix` | Call/Put option matrices |

## Business symbols

- Underlying: `اهرم`
- Calls: `ضهرم`
- Puts: `طهرم`

## Phases implemented

1. Skeleton, config, health, Supabase migrations
2. SourceArena client + raw response storage
3. Option pair matching (>30 days to expiry)
4. Arbitrage engine (R formula)
5. HV engine (40 trading days)
6. Breadth Thrust (10-day avg)
7. Advance/Decline ratio (10-day avg)
8. Call/Put matrices
9. HTML dashboard
10. Telegram alerts with duplicate prevention
