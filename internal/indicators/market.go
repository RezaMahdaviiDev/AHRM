package indicators

import (
	"fmt"
)

type DailyMarket struct {
	Date     string // YYYY-MM-DD; empty when not needed
	Positive int
	Negative int
	Total    int
}

type SymbolRow struct {
	Name      string  `json:"name"`
	ChangePct float64 `json:"change_pct"`
	Status    string  `json:"status"` // "positive" | "negative" | "neutral"
}

type DailyValue struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type IndicatorResult struct {
	CurrentValue float64      `json:"current_value"`
	Average10Day float64      `json:"average_10_day"`
	DaysInWindow int          `json:"days_in_window"`
	AlertState   string       `json:"alert_state"`
	History      []DailyValue `json:"history,omitempty"`
}

type Thresholds struct {
	High float64
	Low  float64
}

func (t Thresholds) AlertState(value float64) string {
	if t.High > 0 && value >= t.High {
		return "high"
	}
	if t.Low > 0 && value <= t.Low {
		return "low"
	}
	return "normal"
}

func AverageLast(values []float64, days int) (float64, error) {
	if days <= 0 {
		return 0, fmt.Errorf("days must be positive")
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("need %d values, got 0", days)
	}
	n := days
	if len(values) < n {
		n = len(values)
	}
	slice := values[len(values)-n:]
	var sum float64
	for _, v := range slice {
		sum += v
	}
	return sum / float64(len(slice)), nil
}

func BreadthDailyValue(day DailyMarket) (float64, error) {
	if day.Total <= 0 {
		return 0, fmt.Errorf("total symbols must be positive")
	}
	return float64(day.Positive) / float64(day.Total), nil
}

func AdvanceDeclineDailyValue(day DailyMarket) (float64, error) {
	if day.Positive <= 0 && day.Negative <= 0 {
		return 0, fmt.Errorf("no advancing or declining symbols")
	}
	denom := day.Negative
	if denom <= 0 {
		denom = 1
	}
	return float64(day.Positive) / float64(denom), nil
}

type BreadthEngine struct {
	Thresholds Thresholds
	Window     int
}

func NewBreadthEngine(th Thresholds) *BreadthEngine {
	return &BreadthEngine{Thresholds: th, Window: 10}
}

func (e *BreadthEngine) Evaluate(history []DailyMarket) (IndicatorResult, error) {
	values := make([]float64, 0, len(history))
	for _, day := range history {
		v, err := BreadthDailyValue(day)
		if err != nil {
			return IndicatorResult{}, err
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return IndicatorResult{}, fmt.Errorf("empty history")
	}
	avg, err := AverageLast(values, e.Window)
	if err != nil {
		return IndicatorResult{}, err
	}
	current := values[len(values)-1]
	n := e.Window
	if len(values) < n {
		n = len(values)
	}
	return IndicatorResult{
		CurrentValue: current,
		Average10Day: avg,
		DaysInWindow: n,
		AlertState:   e.Thresholds.AlertState(avg),
	}, nil
}

type AdvanceDeclineEngine struct {
	Thresholds Thresholds
	Window     int
}

func NewAdvanceDeclineEngine(th Thresholds) *AdvanceDeclineEngine {
	return &AdvanceDeclineEngine{Thresholds: th, Window: 10}
}

func (e *AdvanceDeclineEngine) Evaluate(history []DailyMarket) (IndicatorResult, error) {
	values := make([]float64, 0, len(history))
	for _, day := range history {
		v, err := AdvanceDeclineDailyValue(day)
		if err != nil {
			return IndicatorResult{}, err
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return IndicatorResult{}, fmt.Errorf("empty history")
	}
	avg, err := AverageLast(values, e.Window)
	if err != nil {
		return IndicatorResult{}, err
	}
	current := values[len(values)-1]
	n := e.Window
	if len(values) < n {
		n = len(values)
	}
	return IndicatorResult{
		CurrentValue: current,
		Average10Day: avg,
		DaysInWindow: n,
		AlertState:   e.Thresholds.AlertState(avg),
	}, nil
}
