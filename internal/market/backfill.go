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
)

type dayAgg struct{ positive, negative, total int }

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
