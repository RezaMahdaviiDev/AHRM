package market

import (
	"context"
	"sync"
	"time"

	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

const (
	backfillLookbackDays = 30
	backfillConcurrency  = 3
	backfillCallDelay    = 150 * time.Millisecond

	// backfillRequiredDays is the number of trading days the breadth/advance-decline
	// indicators need (their moving-average window). Once the store already holds this
	// many recent days, the per-symbol candle backfill is unnecessary.
	backfillRequiredDays = 10
	// backfillGateWindowDays is the recent calendar window inspected when deciding
	// whether to skip the backfill. It comfortably contains backfillRequiredDays
	// trading days plus weekends, so a recent multi-day gap drops the count below the
	// requirement and re-triggers the backfill.
	backfillGateWindowDays = 16
)

type dayAgg struct{ positive, negative, total int }

// NeedsBackfill reports whether the expensive per-symbol candle backfill should run.
//
// In steady state the live market snapshot (ClassifyDay + UpsertToday) appends one row
// per trading day, so the indicators' 10-day window can be served straight from SQL.
// This guard returns false (skip) once the store already holds at least
// backfillRequiredDays distinct days within the recent gate window, avoiding the
// hundreds of candle requests the backfill would otherwise issue on every startup and
// every daily run. It returns true on a cold start or when a recent gap has left the
// window short, so the backfill still seeds/repairs history when actually needed.
//
// On a store error it returns true (fail-safe: prefer an extra backfill over a stale
// indicator window).
func NeedsBackfill(ctx context.Context, store DailyStore) (bool, error) {
	if store == nil {
		return false, nil
	}
	to := time.Now()
	from := to.AddDate(0, 0, -backfillGateWindowDays)
	existing, err := store.ExistingDays(ctx, from, to)
	if err != nil {
		return true, err
	}
	return len(existing) < backfillRequiredDays, nil
}

// BackfillHistory fetches per-symbol daily candles for the past backfillLookbackDays calendar
// days and stores the resulting DailyMarket records for each past trading day.
// Today is skipped — today's data comes from the live market API (ClassifyDay).
// Safe to call concurrently; only writes dates not already covered by the store.
func BackfillHistory(ctx context.Context, client *sourcearena.Client,
	symbols []sourcearena.SymbolQuote, store DailyStore) error {

	traded := make([]string, 0, len(symbols))
	for _, s := range symbols {
		if s.TradeValue > 0 {
			traded = append(traded, s.Name)
		}
	}
	if len(traded) == 0 || client == nil {
		return nil
	}

	from := time.Now().AddDate(0, 0, -backfillLookbackDays)
	to := time.Now()

	var mu sync.Mutex
	perDay := map[string]*dayAgg{}

	sem := make(chan struct{}, backfillConcurrency)
	var wg sync.WaitGroup

	for _, name := range traded {
		select {
		case <-ctx.Done():
			break
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		time.Sleep(backfillCallDelay)
		go func(sym string) {
			defer wg.Done()
			defer func() { <-sem }()

			candles, err := client.FetchCandles(ctx, sourcearena.CandleRequest{
				Symbol:     sym,
				From:       from,
				To:         to,
				Resolution: sourcearena.Resolution1D,
				Type:       sourcearena.AdjustCapAndDividend,
			})
			if err != nil || len(candles) < 2 {
				return
			}

			mu.Lock()
			for i := 1; i < len(candles); i++ {
				prev := candles[i-1]
				if prev.Close <= 0 || candles[i].Time <= 0 {
					continue
				}
				dateStr := time.Unix(candles[i].Time, 0).UTC().Format("2006-01-02")
				changePct := (candles[i].Close - prev.Close) / prev.Close * 100
				agg := perDay[dateStr]
				if agg == nil {
					agg = &dayAgg{}
					perDay[dateStr] = agg
				}
				agg.total++
				switch {
				case changePct > 0.5:
					agg.positive++
				case changePct < -0.5:
					agg.negative++
				}
			}
			mu.Unlock()
		}(name)
	}
	wg.Wait()

	today := time.Now().UTC().Format("2006-01-02")
	for dateStr, agg := range perDay {
		if dateStr == today {
			continue
		}
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		_ = store.UpsertDay(ctx, t, indicators.DailyMarket{
			Positive: agg.positive,
			Negative: agg.negative,
			Total:    agg.total,
		})
	}
	return nil
}
