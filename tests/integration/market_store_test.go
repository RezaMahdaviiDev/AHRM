package integration_test

import (
	"context"
	"testing"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/db"
	"ahrm/internal/indicators"
	"ahrm/internal/market"
)

// TestMarketStoreRoundTrip verifies the production storage path (PostgreSQL) for the
// breadth/advance-decline daily stats: the same market.Store used by the server writes
// one row per day into market_daily_stats and reads the recent window back.
//
// It self-skips unless Supabase/PostgreSQL is configured (e.g. via .env.dev.example +
// `make db-up`). Synthetic far-future dates are used so the inserted rows are always the
// most recent and never collide with real data; they are deleted on cleanup.
func TestMarketStoreRoundTrip(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Supabase.Configured() {
		t.Skip("supabase not configured; skipping market store integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	store := market.NewStore(pool)

	const n = 10
	base := time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)
	dates := make([]time.Time, 0, n)
	// Registered after `defer pool.Close()`, so it runs first (LIFO) while the pool is
	// still open and removes the synthetic rows.
	defer func() {
		for _, d := range dates {
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM market_daily_stats WHERE day = $1`,
				d.UTC().Truncate(24*time.Hour))
		}
	}()
	for i := 0; i < n; i++ {
		d := base.AddDate(0, 0, i)
		dates = append(dates, d)
		// Positive increases with the date so we can assert ordering on read-back.
		if err := store.UpsertDay(ctx, d, indicators.DailyMarket{
			Positive: 60 + i,
			Negative: 20,
			Total:    100,
		}); err != nil {
			t.Fatalf("UpsertDay(%s) error = %v", d.Format("2006-01-02"), err)
		}
	}

	hist, err := store.LastDays(ctx, n)
	if err != nil {
		t.Fatalf("LastDays() error = %v", err)
	}
	if len(hist) != n {
		t.Fatalf("LastDays() returned %d rows, want %d", len(hist), n)
	}
	// LastDays returns oldest-first, so the final element is the newest day.
	if got := hist[n-1].Positive; got != 60+n-1 {
		t.Fatalf("newest day Positive = %d, want %d", got, 60+n-1)
	}

	// The stored window must feed the breadth indicator (the original purpose).
	breadth := indicators.NewBreadthEngine(indicators.Thresholds{High: 0.618, Low: 0.4})
	res, err := breadth.Evaluate(hist)
	if err != nil {
		t.Fatalf("breadth Evaluate() error = %v", err)
	}
	if res.DaysInWindow != n {
		t.Fatalf("breadth DaysInWindow = %d, want %d", res.DaysInWindow, n)
	}
}
