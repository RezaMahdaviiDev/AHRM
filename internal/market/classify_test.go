package market_test

import (
	"testing"

	"ahrm/internal/market"
	"ahrm/internal/sourcearena"
)

func TestClassifyDay(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "a", FinalPriceChangePct: 1, TradeValue: 100},    // positive (> 0.5)
		{Name: "b", FinalPriceChangePct: -2, TradeValue: 100},   // negative (< -0.5)
		{Name: "c", FinalPriceChangePct: 0.3, TradeValue: 100},  // neutral (-0.5 to +0.5)
		{Name: "d", FinalPriceChangePct: 3, TradeValue: 100},    // positive (> 0.5)
		{Name: "e", FinalPriceChangePct: 5, TradeValue: 0},      // not traded — excluded
	}
	day := market.ClassifyDay(symbols)
	// Positive: a, d = 2; Negative: b = 1; Neutral: c = 1; Total traded: 4
	if day.Positive != 2 || day.Negative != 1 || day.Total != 4 {
		t.Fatalf("day=%+v", day)
	}
}

func TestClassifyDayExcludesNonStockMarkets(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "فملی", Market: "بازار اول (تابلوی اصلی) بورس", FinalPriceChangePct: 1, TradeValue: 100},
		{Name: "اهرم", Market: "بازار صندوق های قابل معامله", FinalPriceChangePct: 2, TradeValue: 100},
		{Name: "عیار", Market: "صندوق های کالایی", FinalPriceChangePct: -1, TradeValue: 100},
		{Name: "آلا", Market: "بازار ابزارهاي نوين مالي فرابورس", FinalPriceChangePct: 1, TradeValue: 100},
		{Name: "رشد", Market: "بازار نوآفرین - رشد", FinalPriceChangePct: 1, TradeValue: 100},
		{Name: "دانش", Market: "بازار نوآفرین - دانش بنیان", FinalPriceChangePct: 1, TradeValue: 100},
		{Name: "کالا", Market: "بورس کالا", FinalPriceChangePct: 1, TradeValue: 100},
	}
	day := market.ClassifyDay(symbols)
	// only فملی passes; all ETF/fund/non-stock markets excluded
	if day.Total != 1 || day.Positive != 1 {
		t.Fatalf("expected total=1 positive=1, got %+v", day)
	}
}

func TestClassifyDayExcludesHaqTaqadom(t *testing.T) {
	symbols := []sourcearena.SymbolQuote{
		{Name: "فملی", Market: "بازار اول (تابلوی اصلی) بورس", FinalPriceChangePct: 1, TradeValue: 100},
		{Name: "فملیح", Market: "-", FinalPriceChangePct: 1, TradeValue: 100}, // حق تقدم — excluded
		{Name: "وبملح", Market: "-", FinalPriceChangePct: 2, TradeValue: 100}, // حق تقدم — excluded
	}
	day := market.ClassifyDay(symbols)
	if day.Total != 1 || day.Positive != 1 {
		t.Fatalf("expected total=1 (only فملی), got %+v", day)
	}
}
