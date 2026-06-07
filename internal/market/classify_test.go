package market_test

import (
	"testing"

	"ahrm/internal/market"
	"ahrm/internal/sourcearena"
)

func TestClassifyDay(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "a", ClosePriceChangePct: 1},
		{Name: "b", ClosePriceChangePct: -2},
		{Name: "c", ClosePriceChangePct: 0},
		{Name: "d", ClosePriceChangePct: 3},
	}
	day := market.ClassifyDay(symbols)
	if day.Positive != 2 || day.Negative != 1 || day.Total != 4 {
		t.Fatalf("day=%+v", day)
	}
}
