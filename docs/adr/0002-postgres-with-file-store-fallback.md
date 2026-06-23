# PostgreSQL in production, SQLite as the local fallback

Persistent state (market daily stats, alert history, raw API responses) lives in
PostgreSQL (Supabase in production), accessed via `pgx` with SQL migrations applied on
startup. When Supabase is **not** configured, the scanner transparently falls back to an
embedded **SQLite** database (`data/market.db`, `internal/market/sqlitestore.go`) selected
behind the `market.DailyStore` interface in `cmd/server/main.go`.

The SQLite fallback is deliberate: it gives local development and CI a real, persistent SQL
store with zero setup (no Docker, no server) while production keeps managed Postgres. The
cost is two store implementations of `market.DailyStore` to keep in sync. (An older JSON
file store, `internal/market/filestore.go`, predates the SQLite store and is no longer
wired in.)
