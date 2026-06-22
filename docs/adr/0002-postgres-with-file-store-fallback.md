# PostgreSQL for persistence, with a JSON file-store fallback

Persistent state (market daily stats, alert history, raw API responses) lives in
PostgreSQL (Supabase in production), accessed via `pgx` with SQL migrations applied on
startup. When Supabase is not configured, the scanner transparently falls back to a JSON
file store for market history (`internal/market/filestore.go`).

The fallback is deliberate, not a leftover: it lets the app run locally and in CI with no
database, keeping the dev loop frictionless. The cost is two storage implementations of
`market.DailyStore` that must be kept in sync, and the file store is capped at 30 days
with no concurrency guarantees beyond a single process.
