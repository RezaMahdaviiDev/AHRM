# Cold-start backfill gate: banner shown only during actual work

When the market history store has fewer than 10 trading days within the most recent
16-calendar-day window (`NeedsBackfill` in `internal/market/backfill.go`), a backfill
job fetches daily candle data for all ~758 stock-universe symbols and writes historical
`market_daily_stats` rows. This job takes roughly **22 minutes** (38 batches × 20 symbols,
1 s/symbol + 5 s between batches) and is expected behaviour on a fresh or recently-cleared
database.

The `backfilling` atomic flag — which drives the "بک‌فیل در حال اجرا" banner in the UI
and skips the HV-candle and indicators fetches — is now set **after** the `NeedsBackfill`
guard, not before. This means normal restarts (where ≥ 10 days are already in the store)
never show the banner and never skip HV/indicators. The banner is shown only when the
22-minute job actually runs.

**Trade-off accepted**: on a genuine cold start the banner still appears for the full
22-minute duration; there is no incremental progress visible in-page beyond the batch
counter in the JSON snapshot. Adding a live progress bar was skipped (YAGNI — this event
is rare: it fires only on first deploy or after a deliberate DB reset).

**Post-market transient errors**: After the first refresh following backfill completion,
the `indicators` endpoint may return an error if SourceArena is recalculating indicators
post-session. This produces a `WARN snapshot refresh completed with errors count=1` log
entry and self-resolves within one or two refresh cycles. No code change is required; it
is not a sign of the scanner being stuck.
