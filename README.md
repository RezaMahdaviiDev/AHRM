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
