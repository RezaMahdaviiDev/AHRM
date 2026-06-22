package market

import (
	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

func ClassifyDay(symbols []sourcearena.SymbolQuote) indicators.DailyMarket {
	var positive, negative, total int
	for _, sym := range symbols {
		if sym.TradeValue <= 0 || isOptionSymbol(sym.Name) {
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
