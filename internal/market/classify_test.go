package market_test

import (
	"testing"

	"ahrm/internal/market"
	"ahrm/internal/sourcearena"
)

func TestClassifyDay(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "a", ClosePriceChangePct: 1, TradeValue: 100},    // positive (> 0.5)
		{Name: "b", ClosePriceChangePct: -2, TradeValue: 100},   // negative (< -0.5)
		{Name: "c", ClosePriceChangePct: 0.3, TradeValue: 100},  // neutral (-0.5 to +0.5)
		{Name: "d", ClosePriceChangePct: 3, TradeValue: 100},    // positive (> 0.5)
		{Name: "e", ClosePriceChangePct: 5, TradeValue: 0},      // not traded — excluded
	}
	day := market.ClassifyDay(symbols)
	// Positive: a, d = 2; Negative: b = 1; Neutral: c = 1; Total traded: 4
	if day.Positive != 2 || day.Negative != 1 || day.Total != 4 {
		t.Fatalf("day=%+v", day)
	}
}
