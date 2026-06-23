package market

import (
	"sort"

	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

// excludedMarkets lists TSE/OTC market categories that are not joint-stock companies.
// ETFs, commodity funds, and other non-equity instruments are excluded from breadth
// calculations so that only شرکت‌های سهامی are counted.
var excludedMarkets = map[string]struct{}{
	"بازار صندوق های قابل معامله":      {},
	"صندوق های قابل معامله":            {},
	"صندوق های کالایی":                 {},
	"بازار ابزارهاي نوين مالي فرابورس": {},
	"بازار نوآفرین - رشد":              {},
	"بازار نوآفرین - دانش بنیان":       {},
	"بورس کالا":                        {},
}

func ClassifyDay(symbols []sourcearena.SymbolQuote) indicators.DailyMarket {
	var positive, negative, total int
	for _, sym := range symbols {
		if sym.TradeValue <= 0 || isOptionSymbol(sym.Name) || isHaqTaqadom(sym.Name) {
			continue
		}
		if _, excluded := excludedMarkets[sym.Market]; excluded {
			continue
		}
		total++
		switch {
		case sym.ClosePriceChangePct > 0.5:
			positive++
		case sym.ClosePriceChangePct < -0.5:
			negative++
		}
	}
	return indicators.DailyMarket{
		Positive: positive,
		Negative: negative,
		Total:    total,
	}
}

// SymbolRows returns the filtered set of symbols (same universe as ClassifyDay)
// with each symbol's change percent and positive/negative/neutral status.
// Results are sorted descending by ChangePct (best performers first).
func SymbolRows(symbols []sourcearena.SymbolQuote) []indicators.SymbolRow {
	out := make([]indicators.SymbolRow, 0, len(symbols))
	for _, sym := range symbols {
		if sym.TradeValue <= 0 || isOptionSymbol(sym.Name) || isHaqTaqadom(sym.Name) {
			continue
		}
		if _, excluded := excludedMarkets[sym.Market]; excluded {
			continue
		}
		status := "neutral"
		switch {
		case sym.ClosePriceChangePct > 0.5:
			status = "positive"
		case sym.ClosePriceChangePct < -0.5:
			status = "negative"
		}
		out = append(out, indicators.SymbolRow{
			Name:      sym.Name,
			ChangePct: sym.ClosePriceChangePct,
			Status:    status,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChangePct > out[j].ChangePct })
	return out
}
