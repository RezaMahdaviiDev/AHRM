package market

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

const (
	backfillLookbackDays = 30
	backfillConcurrency  = 1
	backfillCallDelay    = 1 * time.Second
	backfillBatchSize    = 20
	backfillBatchPause   = 5 * time.Second

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

// BackfillHistory fetches per-symbol daily candles for the past backfillLookbackDays
// calendar days and stores the resulting DailyMarket records for each past trading day.
// Today is skipped — today's data comes from the live market API (ClassifyDay).
//
// Symbols are processed in batches of backfillBatchSize. After each batch the
// aggregated day stats are written to the store immediately, so progress is preserved
// if the context is cancelled mid-way. A pause of backfillBatchPause is inserted
// between batches to avoid hammering the API.
// ProgressFunc is called after each batch with (currentBatch, totalBatches, symbolsDone, totalSymbols).
type ProgressFunc func(batch, totalBatches, symbolsDone, totalSymbols int)

func BackfillHistory(ctx context.Context, client *sourcearena.Client,
	symbols []sourcearena.SymbolQuote, store DailyStore, onProgress ProgressFunc) error {

	traded := make([]string, 0, len(symbols))
	for _, s := range symbols {
		if s.TradeValue <= 0 || isOptionSymbol(s.Name) || isHaqTaqadom(s.Name) {
			continue
		}
		if _, excluded := excludedMarkets[s.Market]; excluded {
			continue
		}
		traded = append(traded, s.Name)
	}
	if len(traded) == 0 || client == nil {
		return nil
	}

	from := time.Now().AddDate(0, 0, -backfillLookbackDays)
	to := time.Now()
	today := time.Now().UTC().Format("2006-01-02")

	// allDays accumulates ALL batches before writing — each day's totals
	// reflect the full symbol universe, not just the last batch processed.
	allDays := map[string]*dayAgg{}

	total := len(traded)
	totalBatches := (total + backfillBatchSize - 1) / backfillBatchSize
	if onProgress != nil {
		onProgress(0, totalBatches, 0, total)
	}

	for batchStart := 0; batchStart < total; batchStart += backfillBatchSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		batchEnd := batchStart + backfillBatchSize
		if batchEnd > total {
			batchEnd = total
		}
		batch := traded[batchStart:batchEnd]

		slog.Info("backfill batch started", "batch", batchStart/backfillBatchSize+1, "symbols", len(batch), "progress", batchEnd, "total", total)

		perDay := processBatch(ctx, client, batch, from, to)

		for dateStr, agg := range perDay {
			if dateStr == today {
				continue
			}
			existing := allDays[dateStr]
			if existing == nil {
				existing = &dayAgg{}
				allDays[dateStr] = existing
			}
			existing.positive += agg.positive
			existing.negative += agg.negative
			existing.total += agg.total
		}

		batchNum := batchStart/backfillBatchSize + 1
		slog.Info("backfill batch done", "batch", batchNum, "days_in_batch", len(perDay), "days_accumulated", len(allDays))
		if onProgress != nil {
			onProgress(batchNum, totalBatches, batchEnd, total)
		}

		if batchEnd < total {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backfillBatchPause):
			}
		}
	}

	// Write final accumulated results once all batches are done.
	daysWritten := 0
	for dateStr, agg := range allDays {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		_ = store.UpsertDay(ctx, t, indicators.DailyMarket{
			Positive: agg.positive,
			Negative: agg.negative,
			Total:    agg.total,
		})
		daysWritten++
	}
	slog.Info("backfill complete", "days_written", daysWritten)
	return nil
}

func processBatch(ctx context.Context, client *sourcearena.Client,
	symbols []string, from, to time.Time) map[string]*dayAgg {

	var mu sync.Mutex
	perDay := map[string]*dayAgg{}

	sem := make(chan struct{}, backfillConcurrency)
	var wg sync.WaitGroup

	for _, name := range symbols {
		if ctx.Err() != nil {
			break
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
			if err != nil {
				slog.Debug("backfill candle error", "sym", sym, "err", err)
				return
			}
			if len(candles) < 2 {
				slog.Debug("backfill candle too few", "sym", sym, "count", len(candles))
				return
			}
			slog.Debug("backfill candle ok", "sym", sym, "count", len(candles), "first_time", candles[0].Time)

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
	return perDay
}

// isOptionSymbol returns true for TSE derivative/option symbols.
// Call options start with ض and put options start with ط, but both
// contain digits (e.g. ضملت4019, طهرم5027). Regular equity symbols
// are pure Persian text with no digits, so a digit check is sufficient.
func isOptionSymbol(name string) bool {
	for _, r := range name {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// isHaqTaqadom returns true for حق تقدم (preemptive rights) symbols.
// By TSE convention these symbols end with the letter ح (e.g. وبملح, فملیح).
func isHaqTaqadom(name string) bool {
	runes := []rune(name)
	return len(runes) >= 2 && runes[len(runes)-1] == 'ح'
}
