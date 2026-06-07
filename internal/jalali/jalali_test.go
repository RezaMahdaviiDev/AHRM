package jalali_test

import (
	"testing"
	"time"

	"ahrm/internal/jalali"
)

func TestParseDate(t *testing.T) {
	got, err := jalali.ParseDate("1404/03/15")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	if got.IsZero() {
		t.Fatal("expected non-zero time")
	}
}

func TestCalendarDaysUntil(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	if got := jalali.CalendarDaysUntil(from, to); got != 31 {
		t.Fatalf("CalendarDaysUntil() = %d, want 31", got)
	}
}
