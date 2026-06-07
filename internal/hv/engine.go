package hv

import (
	"fmt"
	"math"

	"ahrm/internal/sourcearena"
)

const (
	TradingDays  = 40
	HVMultiplier = 15.8
)

type Result struct {
	HVPct            float64 `json:"hv_pct"`
	DailyVolatility  float64 `json:"daily_volatility"`
	SampleSize       int     `json:"sample_size"`
}

type Engine struct {
	TradingDays  int
	HVMultiplier float64
}

func NewEngine() *Engine {
	return &Engine{TradingDays: TradingDays, HVMultiplier: HVMultiplier}
}

func (e *Engine) Calculate(candles []sourcearena.Candle) (Result, error) {
	if e.TradingDays <= 0 {
		e.TradingDays = TradingDays
	}
	if e.HVMultiplier <= 0 {
		e.HVMultiplier = HVMultiplier
	}
	if len(candles) < e.TradingDays+1 {
		return Result{}, fmt.Errorf("need at least %d candles, got %d", e.TradingDays+1, len(candles))
	}
	recent := candles[len(candles)-e.TradingDays-1:]
	returns := make([]float64, 0, e.TradingDays)
	for i := 1; i < len(recent); i++ {
		prev := recent[i-1].Close
		cur := recent[i].Close
		if prev <= 0 || cur <= 0 {
			return Result{}, fmt.Errorf("invalid candle price at index %d", i)
		}
		returns = append(returns, math.Log(cur/prev))
	}
	dailyVol := stdDev(returns)
	hv := dailyVol * e.HVMultiplier * 100
	return Result{
		HVPct:           hv,
		DailyVolatility: dailyVol,
		SampleSize:      len(returns),
	}, nil
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}
