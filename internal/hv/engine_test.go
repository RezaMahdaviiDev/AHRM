package hv_test

import (
	"testing"

	"ahrm/internal/hv"
	"ahrm/internal/sourcearena"
)

func makeCandles(start, step float64, n int) []sourcearena.Candle {
	out := make([]sourcearena.Candle, n)
	price := start
	for i := range out {
		out[i] = sourcearena.Candle{Close: price}
		price += step
	}
	return out
}

func TestCalculateHV(t *testing.T) {
	engine := hv.NewEngine()
	candles := makeCandles(100, 1, 42)
	result, err := engine.Calculate(candles)
	if err != nil {
		t.Fatal(err)
	}
	if result.SampleSize != 40 {
		t.Fatalf("sample=%d", result.SampleSize)
	}
	if result.HVPct <= 0 {
		t.Fatalf("hv=%v", result.HVPct)
	}
	if len(result.Series) != 41 {
		t.Fatalf("series len=%d", len(result.Series))
	}
}

func TestLogReturnStdDev(t *testing.T) {
	engine := hv.NewEngine()
	candles := []sourcearena.Candle{{Close: 100}, {Close: 110}, {Close: 105}}
	engine.TradingDays = 2
	result, err := engine.Calculate(candles)
	if err != nil {
		t.Fatal(err)
	}
	if result.DailyVolatility <= 0 {
		t.Fatalf("daily vol=%v", result.DailyVolatility)
	}
}

func TestCalculateInsufficientData(t *testing.T) {
	engine := hv.NewEngine()
	_, err := engine.Calculate([]sourcearena.Candle{{Close: 100}})
	if err == nil {
		t.Fatal("expected error")
	}
}
