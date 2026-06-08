package arbitrage_test

import (
	"testing"

	"ahrm/internal/arbitrage"
	"ahrm/internal/pairs"
	"ahrm/internal/sourcearena"
)

func TestCalculateFormula(t *testing.T) {
	engine := arbitrage.NewEngine()
	pair := pairs.Pair{
		Call: sourcearena.Option{Name: "ضهرم1200", ClosePrice: 1500, RealSellVolume: 250000},
		Put:  sourcearena.Option{ClosePrice: 800},
		Strike:      12000,
		ExpiryLabel: "1404/09/15",
	}
	opp, err := engine.Calculate(pair, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if opp.Symbol != "ضهرم1200" || opp.SellVolume != 250000 {
		t.Fatalf("symbol=%q sell_volume=%v", opp.Symbol, opp.SellVolume)
	}
	if opp.Spread != 700 {
		t.Fatalf("spread=%v", opp.Spread)
	}
	if opp.Capital != 24300 {
		t.Fatalf("capital=%v", opp.Capital)
	}
	wantR := ((12000.0 - 24300.0) / 24300.0) * 100
	if diff := opp.ReturnPct - wantR; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("return=%v want=%v", opp.ReturnPct, wantR)
	}
}

func TestCalculateZeroCapital(t *testing.T) {
	engine := arbitrage.NewEngine()
	pair := pairs.Pair{
		Call:   sourcearena.Option{ClosePrice: 30000},
		Put:    sourcearena.Option{ClosePrice: 0},
		Strike: 12000,
	}
	_, err := engine.Calculate(pair, 25000)
	if err == nil {
		t.Fatal("expected error for non-positive capital")
	}
}

func TestCalculateZeroUnderlying(t *testing.T) {
	engine := arbitrage.NewEngine()
	pair := pairs.Pair{
		Call:   sourcearena.Option{ClosePrice: 100},
		Put:    sourcearena.Option{ClosePrice: 50},
		Strike: 12000,
	}
	_, err := engine.Calculate(pair, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}
