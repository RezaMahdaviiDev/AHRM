package scanner

import (
	"testing"

	"ahrm/internal/indicators"
)

func TestMergeTodayIntoHistory(t *testing.T) {
	history := []indicators.DailyMarket{
		{Date: "2026-07-10", Positive: 100, Negative: 50, Total: 200},
		{Date: "2026-07-11", Positive: 10, Negative: 90, Total: 200},
	}
	today := indicators.DailyMarket{Date: "2026-07-12", Positive: 74, Negative: 602, Total: 707}

	got := mergeTodayIntoHistory(history, today)
	if len(got) != 3 {
		t.Fatalf("append: len=%d want 3", len(got))
	}
	if got[2].Positive != 74 {
		t.Fatalf("append: last day=%+v", got[2])
	}

	updated := indicators.DailyMarket{Date: "2026-07-12", Positive: 80, Negative: 600, Total: 707}
	got = mergeTodayIntoHistory(got, updated)
	if len(got) != 3 {
		t.Fatalf("replace: len=%d want 3", len(got))
	}
	if got[2].Positive != 80 {
		t.Fatalf("replace: last day=%+v", got[2])
	}
}
