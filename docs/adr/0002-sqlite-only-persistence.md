# ADR 0002: SQLite-only persistence (Supabase/PostgreSQL removed)

**Updated 2026-06-25** — originally "PostgreSQL in production, SQLite as local fallback."

All persistence (market daily stats, alert history) is now handled exclusively by
**SQLite** (`data/market.db`, `data/alerts.db`). PostgreSQL/Supabase was removed after it
was never deployed; maintaining two store implementations and the `pgx` dependency for an
unused path was not justified.

`market.DailyStore`, `SymbolSnapshotStore`, and `SymbolRegistryStore` interfaces are
defined in `internal/market/interfaces.go` and implemented by `SQLiteStore`
(`internal/market/sqlitestore.go`). Alert dedup is backed by `SQLiteStore` in
`internal/alerts/store.go`. The old JSON file store (`internal/market/filestore.go`)
remains on disk but is not wired in.
