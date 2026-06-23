package market

import (
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
