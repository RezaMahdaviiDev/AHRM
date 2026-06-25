# ADR 0006: Holiday Detection via Stats Comparison

**Decision:** Detect market holidays by comparing today's computed breadth stats
(Positive/Negative/Total) with the most recent record in `market_daily_stats`. If they
are identical, skip recording — no calendar or API-level holiday flag is needed.

**Why:** SourceArena caches the last trading day's full symbol snapshot on holidays,
returning non-zero `TradeValue` for all symbols. This means `ClassifyDay` produces the
same counts as the last real session, making identical stats a reliable proxy for "market
closed today." A proper TSE holiday calendar was rejected because it would need constant
maintenance and the stats-comparison approach has no operational overhead.

**Trade-off:** Two consecutive trading days with the exact same Positive/Negative/Total
counts would cause the second day to be skipped. In practice this is essentially
impossible given 700+ symbols with daily price movement.
