# AGENTS.md

Operating guidance for AI agents working on AHRM — a Go service that scans the Iranian
`اهرم` options chain, computes arbitrage / volatility / breadth signals, and pushes
Bale alerts.

## Living documentation lifecycle

This repo keeps a small set of **living documents** that exist to make agents faster and
more accurate. Treat them as part of the work, not an afterthought: read them **before**
you start, and update them **after** you change things.

- `CONTEXT.md` (repo root) — the domain glossary / ubiquitous language. The fastest way to
  understand what `اهرم`/`ضهرم`/`طهرم`, "Return (R)", "Static ROI", "Breadth Thrust", etc.
  actually mean, and which synonyms to avoid.
- `docs/adr/` — Architecture Decision Records: short notes on hard-to-reverse, surprising
  decisions (PostgreSQL prod + SQLite local default, why SourceArena needs Iranian egress,
  why HV uses a 180-day window, …).
- `.claude/skills/` — reusable engineering SOPs (auto-discovered by Cursor and Claude
  Code). See `.claude/skills/README.md`.

### At the start of a task

1. Read `CONTEXT.md` to load the domain vocabulary.
2. Skim `docs/adr/` for any ADR touching the area you're about to change.
3. Use the vocabulary from `CONTEXT.md` consistently in code, comments, and messages.

### At the end of a task (before you finish / open a PR)

Update the living docs **in the same change** so they never drift:

1. **Glossary** — if you introduced, renamed, or sharpened a domain term, update
   `CONTEXT.md`. Keep it a tight glossary (1–2 sentences per term, an `_Avoid_` list for
   rejected synonyms); never put implementation details, formulas, or specs in it.
2. **Decisions** — if you made a decision that is *hard to reverse*, *surprising without
   context*, **and** the result of a *real trade-off*, add the next-numbered ADR in
   `docs/adr/` (a single paragraph is fine). If none of those three hold, do not add one.
3. Keep both accurate: if a change contradicts an existing term or ADR, fix the doc rather
   than leaving it stale.

### How to do the doc work

Use the installed skills rather than reinventing the process:

- `domain-modeling` — the active discipline for building/sharpening `CONTEXT.md` and ADRs
  (it owns the exact `CONTEXT.md` and ADR formats).
- `grill-with-docs` — when aligning on a new change against a codebase; it records
  terminology and decisions into `CONTEXT.md`/ADRs as you go.
- `codebase-design`, `tdd`, `diagnosing-bugs` — design/test/debug discipline for the work
  itself.

## Build / test / run

Standard commands live in the `Makefile` and `README.md` — prefer those.

- `make run` — start the server on `:8080`. **No external services are required**: when
  Supabase is not configured the scanner persists market history to `data/market.db` and
  alert history to `data/alerts.db` (both SQLite). `/health` (liveness) and `/ready`
  (readiness JSON) need no external
  deps and are the quickest way to confirm the server is up.
- `make test` / `make build` — unit tests and binary build. The suite is expected to be
  green; CI (`.github/workflows/ci.yml`) runs `go vet` + `go build` + `go test ./...` on
  every PR using the Go version in `go.mod`.
- `make test-integration` — needs a configured PostgreSQL/Supabase, else self-skips.

## Cursor Cloud specific instructions

Durable, non-obvious notes for cloud agents (the dependency-refresh `go mod download` is
handled by the startup update script — don't add it here):

- **The toolchain is Go 1.25** (`go.mod`). The system Go may be older; the Go toolchain
  auto-downloads the right version on first build, so the initial `go build`/`go test` can
  be slow while it fetches.
- **`.env` is optional.** `config.Load()` uses `godotenv` and silently ignores a missing
  `.env`, falling back to defaults in `internal/config/config.go`. Copy `.env.example` to
  `.env` only when you need to set tokens/DB creds.
- **Live market data requires `SOURCEARENA_API_TOKEN` *and* Iranian network egress**
  (directly or via `SOURCEARENA_HTTP_PROXY`). From outside Iran the SourceArena API is
  unreachable, so the dashboard, `/arbitrage`, `/hv`, `/market`, `/matrix` pages render but
  show zeros and a "sourcearena client not configured / unreachable" notice. This is
  expected in the cloud VM and is not a setup failure. See `docs/adr/0003-*`.
- **The UI is RTL/Persian.**
- **Lint:** `go vet ./...` is clean; `gofmt -l .` reports several pre-existing unformatted
  files — do not reformat them unless that is the task.
