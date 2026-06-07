package market

import (
	"ahrm/internal/indicators"
	"ahrm/internal/sourcearena"
)

func ClassifyDay(symbols []sourcearena.SymbolQuote) indicators.DailyMarket {
	var positive, negative int
	for _, symbol := range symbols {
		switch {
		case symbol.ClosePriceChangePct > 0:
			positive++
		case symbol.ClosePriceChangePct < 0:
			negative++
		}
	}
	return indicators.DailyMarket{
		Positive: positive,
		Negative: negative,
		Total:    len(symbols),
	}
}
