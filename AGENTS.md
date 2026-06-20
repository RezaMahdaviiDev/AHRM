# AGENTS.md

## Cursor Cloud specific instructions

This is a single-binary Go monolith (`ahrm`) for options-arbitrage scanning, market
indicators, an HTML dashboard, and Telegram/Bale alerts. Standard commands live in the
`Makefile` and `README.md`; prefer those instead of re-deriving them.

### Running

- The server runs with **no external services required**. When Supabase is disabled
  (the default), it falls back to a JSON file store (`data/market_history.json`), so
  `make run` works out of the box and listens on `:8080` (`HTTP_ADDR`).
- A `.env` file is **optional**. `config.Load()` uses `godotenv` and silently ignores a
  missing `.env`, falling back to defaults in `internal/config/config.go`. Copy
  `.env.example` to `.env` only when you need to set tokens/DB creds.
- Live market data requires `SOURCEARENA_API_TOKEN` **and Iranian network egress**
  (directly or via `SOURCEARENA_HTTP_PROXY`). From outside Iran the SourceArena API is
  unreachable, so the dashboard, `/arbitrage`, `/hv`, `/market`, `/matrix` pages render
  but show zeros and a "sourcearena client not configured / unreachable" notice. This is
  expected in the cloud VM and is not a setup failure.
- The UI is RTL/Persian. `/health` (liveness) and `/ready` (readiness JSON) need no
  external deps and are the quickest way to confirm the server is up.

### Tests / lint / build

- `make test` (`go test ./...`): as of this writing one pre-existing test fails,
  `ahrm/internal/ivcalc TestCalculateAllCollectsPerOptionErrors` (a logic test using
  in-memory data; unrelated to environment). All other packages pass.
- `make test-integration` only does real work when Supabase env vars are set; otherwise
  `TestDatabaseConnection` self-skips.
- Lint: `go vet ./...` is clean. `gofmt -l .` reports several pre-existing unformatted
  files — do not reformat them unless that is the task.
- `make build` outputs `bin/server`.
