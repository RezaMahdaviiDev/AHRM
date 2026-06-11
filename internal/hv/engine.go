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

type SeriesPoint struct {
	Time   int64   `json:"t"`
	Close  float64 `json:"close"`
	LogRet float64 `json:"log_ret,omitempty"`
}

type Result struct {
	HVPct           float64       `json:"hv_pct"`
	DailyVolatility float64       `json:"daily_volatility"`
	SampleSize      int           `json:"sample_size"`
	TradingDays     int           `json:"trading_days"`
	HVMultiplier    float64       `json:"hv_multiplier"`
	Series          []SeriesPoint `json:"series,omitempty"`
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
	series := make([]SeriesPoint, 0, len(recent))
	for i, candle := range recent {
		if candle.Close <= 0 {
			return Result{}, fmt.Errorf("invalid candle price at index %d", i)
		}
		point := SeriesPoint{Time: candle.Time, Close: candle.Close}
		if i > 0 {
			prev := recent[i-1].Close
			logRet := math.Log(candle.Close / prev)
			returns = append(returns, logRet)
			point.LogRet = logRet
		}
		series = append(series, point)
	}
	dailyVol := stdDev(returns)
	hv := dailyVol * e.HVMultiplier * 100
	return Result{
		HVPct:           hv,
		DailyVolatility: dailyVol,
		SampleSize:      len(returns),
		TradingDays:     e.TradingDays,
		HVMultiplier:    e.HVMultiplier,
		Series:          series,
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
