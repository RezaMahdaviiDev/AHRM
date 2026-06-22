# AGENTS.md

Operating guidance for AI agents working on AHRM.

## Living documentation lifecycle

This repo keeps a small set of **living documents** that exist to make agents faster and
more accurate. Treat them as part of the work, not an afterthought: read them **before**
you start, and update them **after** you change things.

- `CONTEXT.md` (repo root) — the domain glossary / ubiquitous language. The fastest way to
  understand what `اهرم`/`ضهرم`/`طهرم`, "Return (R)", "Static ROI", "Breadth Thrust", etc.
  actually mean, and which synonyms to avoid.
- `docs/adr/` — Architecture Decision Records: short notes on hard-to-reverse, surprising
  decisions (why a JSON file-store fallback exists, why SourceArena needs Iranian egress,
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

Standard commands live in the `Makefile` and `README.md` — prefer those. In short:
`make run` (server on `:8080`, no external services required — falls back to a JSON file
store), `make test`, `make test-integration` (needs a configured PostgreSQL/Supabase, else
self-skips), and `make build`.
