package indicators_test

import (
	"math"
	"testing"

	"ahrm/internal/indicators"
)

func TestBreadthAverage(t *testing.T) {
	engine := indicators.NewBreadthEngine(indicators.Thresholds{High: 0.59, Low: 0.4})
	history := make([]indicators.DailyMarket, 10)
	for i := range history {
		history[i] = indicators.DailyMarket{Positive: 600, Total: 1000, Negative: 400}
	}
	result, err := engine.Evaluate(history)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(result.Average10Day-0.6) > 0.0001 {
		t.Fatalf("avg=%v", result.Average10Day)
	}
	if result.AlertState != "high" {
		t.Fatalf("alert=%q", result.AlertState)
	}
}

func TestAdvanceDeclineRatio(t *testing.T) {
	engine := indicators.NewAdvanceDeclineEngine(indicators.Thresholds{High: 2.0, Low: 0.8})
	history := []indicators.DailyMarket{
		{Positive: 100, Negative: 50, Total: 150},
		{Positive: 120, Negative: 60, Total: 180},
		{Positive: 80, Negative: 40, Total: 120},
		{Positive: 90, Negative: 45, Total: 135},
		{Positive: 110, Negative: 55, Total: 165},
		{Positive: 100, Negative: 50, Total: 150},
		{Positive: 130, Negative: 65, Total: 195},
		{Positive: 140, Negative: 70, Total: 210},
		{Positive: 150, Negative: 75, Total: 225},
		{Positive: 160, Negative: 80, Total: 240},
	}
	result, err := engine.Evaluate(history)
	if err != nil {
		t.Fatal(err)
	}
	if result.Average10Day != 2.0 {
		t.Fatalf("avg=%v", result.Average10Day)
	}
}

func TestBreadthPartialHistory(t *testing.T) {
	engine := indicators.NewBreadthEngine(indicators.Thresholds{High: 0.59, Low: 0.4})
	history := []indicators.DailyMarket{{Positive: 600, Total: 1000, Negative: 400}}
	result, err := engine.Evaluate(history)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentValue != 0.6 {
		t.Fatalf("current=%v", result.CurrentValue)
	}
	if result.Average10Day != 0.6 {
		t.Fatalf("avg=%v", result.Average10Day)
	}
}

func TestAdvanceDeclineNoDecliners(t *testing.T) {
	engine := indicators.NewAdvanceDeclineEngine(indicators.Thresholds{High: 2.0, Low: 0.8})
	history := []indicators.DailyMarket{{Positive: 800, Negative: 0, Total: 1000}}
	result, err := engine.Evaluate(history)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentValue != 800 {
		t.Fatalf("current=%v", result.CurrentValue)
	}
}

func TestAdvanceDeclineAverageUsesAggregateNotMeanOfRatios(t *testing.T) {
	engine := indicators.NewAdvanceDeclineEngine(indicators.Thresholds{High: 1.4, Low: 0.6})
	// Nine balanced days + one extreme (659/36 ≈ 18) must not push avg to "high"
	// via mean-of-ratios; aggregate sumP/sumN stays near 1.
	history := make([]indicators.DailyMarket, 9)
	for i := range history {
		history[i] = indicators.DailyMarket{Positive: 350, Negative: 350, Total: 700}
	}
	history = append(history, indicators.DailyMarket{Positive: 659, Negative: 36, Total: 731})
	result, err := engine.Evaluate(history)
	if err != nil {
		t.Fatal(err)
	}
	want := float64(350*9+659) / float64(350*9+36)
	if math.Abs(result.Average10Day-want) > 0.0001 {
		t.Fatalf("avg=%v want=%v", result.Average10Day, want)
	}
	if result.AlertState != "normal" {
		t.Fatalf("alert=%q (mean-of-ratios would have been high)", result.AlertState)
	}
}

func TestThresholdValidation(t *testing.T) {
	th := indicators.Thresholds{High: 1.5, Low: 0.5}
	if th.AlertState(1.6) != "high" {
		t.Fatal("expected high")
	}
	if th.AlertState(0.4) != "low" {
		t.Fatal("expected low")
	}
	if th.AlertState(1.0) != "normal" {
		t.Fatal("expected normal")
	}
}
