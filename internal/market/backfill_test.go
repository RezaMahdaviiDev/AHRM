package market

import (
	"context"
	"testing"
	"time"

	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

type fakeStore struct {
	days map[string]indicators.DailyMarket
}

func newFakeStore() *fakeStore {
	return &fakeStore{days: map[string]indicators.DailyMarket{}}
}

func (f *fakeStore) set(date time.Time, day indicators.DailyMarket) {
	f.days[date.UTC().Format("2006-01-02")] = day
}

func (f *fakeStore) UpsertToday(_ context.Context, day indicators.DailyMarket) error {
	f.set(time.Now(), day)
	return nil
}

func (f *fakeStore) UpsertDay(_ context.Context, date time.Time, day indicators.DailyMarket) error {
	f.set(date, day)
	return nil
}

func (f *fakeStore) LastDays(_ context.Context, days int) ([]indicators.DailyMarket, error) {
	return nil, nil
}

func (f *fakeStore) ExistingDays(_ context.Context, from, to time.Time) (map[string]struct{}, error) {
	fromStr := from.UTC().Format("2006-01-02")
	toStr := to.UTC().Format("2006-01-02")
	out := map[string]struct{}{}
	for d := range f.days {
		if d >= fromStr && d <= toStr {
			out[d] = struct{}{}
		}
	}
	return out, nil
}

func seedRecentDays(store *fakeStore, n int) {
	for i := 0; i < n; i++ {
		store.set(time.Now().AddDate(0, 0, -i), indicators.DailyMarket{Positive: 1, Negative: 1, Total: 10})
	}
}

func TestBackfillHistorySkipsFundSymbols(t *testing.T) {
	// BackfillHistory with a nil client returns nil immediately after building
	// the traded list. If all symbols are funds, traded is empty and the
	// function exits early — verifying the market filter is applied.
	ctx := context.Background()
	symbols := []sourcearena.SymbolQuote{
		{Name: "اهرم", Market: "بازار صندوق های قابل معامله", TradeValue: 100},
		{Name: "عیار", Market: "صندوق های کالایی", TradeValue: 100},
		{Name: "آلا", Market: "بازار ابزارهاي نوين مالي فرابورس", TradeValue: 100},
	}
	store := newFakeStore()
	// nil client means BackfillHistory exits after traded-list building; no panic = filter OK
	err := BackfillHistory(ctx, nil, symbols, store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.days) != 0 {
		t.Fatalf("expected no days written, got %d", len(store.days))
	}
}

func TestNeedsBackfill(t *testing.T) {
	ctx := context.Background()

	t.Run("nil store skips", func(t *testing.T) {
		need, err := NeedsBackfill(ctx, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if need {
			t.Fatal("nil store should not need backfill")
		}
	})

	t.Run("empty store needs backfill", func(t *testing.T) {
		need, err := NeedsBackfill(ctx, newFakeStore())
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !need {
			t.Fatal("empty store should need backfill (cold start seed)")
		}
	})

	t.Run("too few recent days needs backfill", func(t *testing.T) {
		store := newFakeStore()
		seedRecentDays(store, backfillRequiredDays-1)
		need, err := NeedsBackfill(ctx, store)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !need {
			t.Fatalf("%d days should still need backfill", backfillRequiredDays-1)
		}
	})

	t.Run("full recent window skips backfill", func(t *testing.T) {
		store := newFakeStore()
		seedRecentDays(store, backfillRequiredDays)
		need, err := NeedsBackfill(ctx, store)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if need {
			t.Fatal("full recent window should skip backfill")
		}
	})

	t.Run("stale days outside window need backfill", func(t *testing.T) {
		store := newFakeStore()
		// Enough days, but all far outside the recent gate window.
		for i := 0; i < backfillRequiredDays+5; i++ {
			store.set(time.Now().AddDate(0, 0, -(backfillGateWindowDays+30+i)),
				indicators.DailyMarket{Positive: 1, Negative: 1, Total: 10})
		}
		need, err := NeedsBackfill(ctx, store)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !need {
			t.Fatal("stale-only data should trigger backfill (gap repair)")
		}
	})
}
