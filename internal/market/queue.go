package market

import (
	"sort"

	"ahrm/internal/sourcearena"
)

// QueueCandidate is a stock likely to open in buy queue (صف خرید) tomorrow.
// Score 3 = IPO + at limit, 2 = at daily limit, 1 = near limit with buy queue.
type QueueCandidate struct {
	Name      string  `json:"name"`
	Market    string  `json:"market"`
	ChangePct float64 `json:"change_pct"`
	BuyVolume float64 `json:"buy_volume"` // 0 if bulk API does not return queue data
	StreakDays int    `json:"streak_days"` // consecutive trading days in buy queue
	Score     int     `json:"score"`
	IsIPO     bool    `json:"is_ipo"`
}

const (
	atLimitThreshold  = 4.9    // TSE daily upper limit ≈ +5%
	nearLimitThreshold = 3.5   // strong move with confirmed buy queue
	minQueueVolume    = 100_000 // minimum buy queue volume for near-limit filter
)

// ScanQueue filters and scores stocks as سرخطی candidates for tomorrow.
// newSymbols: set of names appearing for the first time today (IPO detection).
// streaks: consecutive days each symbol has been in buy queue (0 = unknown/new).
func ScanQueue(symbols []sourcearena.SymbolQuote, newSymbols map[string]bool, streaks map[string]int) []QueueCandidate {
	var out []QueueCandidate
	for _, sym := range symbols {
		if sym.TradeValue <= 0 || isOptionSymbol(sym.Name) || isHaqTaqadom(sym.Name) {
			continue
		}
		if _, excluded := excludedMarkets[sym.Market]; excluded {
			continue
		}

		atLimit := sym.ClosePriceChangePct >= atLimitThreshold
		// near-limit requires confirmed queue volume to avoid noise
		nearLimit := sym.ClosePriceChangePct >= nearLimitThreshold && sym.BuyRow1Volume >= minQueueVolume

		if !atLimit && !nearLimit {
			continue
		}

		isIPO := newSymbols[sym.Name]
		score := 1
		if atLimit {
			score = 2
		}
		if atLimit && isIPO {
			score = 3
		}

		out = append(out, QueueCandidate{
			Name:       sym.Name,
			Market:     sym.Market,
			ChangePct:  sym.ClosePriceChangePct,
			BuyVolume:  sym.BuyRow1Volume,
			StreakDays: streaks[sym.Name],
			Score:      score,
			IsIPO:      isIPO,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].StreakDays != out[j].StreakDays {
			return out[i].StreakDays > out[j].StreakDays
		}
		if out[i].BuyVolume != out[j].BuyVolume {
			return out[i].BuyVolume > out[j].BuyVolume
		}
		return out[i].ChangePct > out[j].ChangePct
	})
	return out
}
