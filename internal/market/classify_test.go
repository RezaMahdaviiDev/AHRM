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

func TestClassifyDayExcludesNonStockMarkets(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "فملی", Market: "بازار اول (تابلوی اصلی) بورس", ClosePriceChangePct: 1, TradeValue: 100},
		{Name: "اهرم", Market: "بازار صندوق های قابل معامله", ClosePriceChangePct: 2, TradeValue: 100},
		{Name: "عیار", Market: "صندوق های کالایی", ClosePriceChangePct: -1, TradeValue: 100},
		{Name: "آلا", Market: "بازار ابزارهاي نوين مالي فرابورس", ClosePriceChangePct: 1, TradeValue: 100},
		{Name: "رشد", Market: "بازار نوآفرین - رشد", ClosePriceChangePct: 1, TradeValue: 100},
		{Name: "دانش", Market: "بازار نوآفرین - دانش بنیان", ClosePriceChangePct: 1, TradeValue: 100},
		{Name: "کالا", Market: "بورس کالا", ClosePriceChangePct: 1, TradeValue: 100},
	}
	day := market.ClassifyDay(symbols)
	// only فملی passes; all ETF/fund/non-stock markets excluded
	if day.Total != 1 || day.Positive != 1 {
		t.Fatalf("expected total=1 positive=1, got %+v", day)
	}
}
