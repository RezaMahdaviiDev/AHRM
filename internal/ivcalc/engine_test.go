package ivcalc_test

import (
	"testing"
	"time"

	"ahrm/internal/blackscholes"
	"ahrm/internal/ivcalc"
	"ahrm/internal/jalali"
	"ahrm/internal/sourcearena"
)

func TestCalculateAllReturnsIV(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	engine := ivcalc.NewEngine()
	engine.Now = func() time.Time { return now }
	S, K, r, sigma := 25000.0, 12000.0, 0.20, 0.35
	expiry, err := jalali.ParseDate("1405/12/15")
	if err != nil {
		t.Fatal(err)
	}
	days := jalali.CalendarDaysUntil(now, expiry)
	T := float64(days) / 365.0
	price := blackscholes.CallPrice(S, K, T, r, sigma)
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: price, StrikePrice: K, ExpiryDate: "1405/12/15"},
	}
	got, errs := engine.CalculateAll(opts, S, r)
	if len(errs) != 0 {
		t.Fatalf("errs=%v", errs)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if got[0].Symbol != "ضهرم1200" {
		t.Fatalf("symbol=%q", got[0].Symbol)
	}
	if got[0].IVPct < 30 || got[0].IVPct > 40 {
		t.Fatalf("iv_pct=%v want ~35", got[0].IVPct)
	}
}

func TestCalculateAllSkipsShortExpiry(t *testing.T) {
	engine := ivcalc.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1300", ClosePrice: 1200, StrikePrice: 13000, ExpiryDate: "1404/04/01"},
	}
	got, errs := engine.CalculateAll(opts, 25000, 0.20)
	if len(errs) != 0 {
		t.Fatalf("errs=%v", errs)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestCalculateAllIgnoresPuts(t *testing.T) {
	engine := ivcalc.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "طهرم1200", ClosePrice: 800, StrikePrice: 12000, ExpiryDate: "1405/12/15"},
	}
	got, errs := engine.CalculateAll(opts, 25000, 0.20)
	if len(errs) != 0 {
		t.Fatalf("errs=%v", errs)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestCalculateAllSkipsOutOfBoundsPriceSilently(t *testing.T) {
	// A call quoting a price at/above the underlying is outside the no-arbitrage
	// bounds. Illiquid/stale quotes like this are skipped silently (no result and no
	// error) so they don't flood the dashboard's error card.
	engine := ivcalc.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 50000, StrikePrice: 12000, ExpiryDate: "1405/12/15"},
	}
	got, errs := engine.CalculateAll(opts, 25000, 0.20)
	if len(got) != 0 {
		t.Fatalf("expected 0 results, got %d", len(got))
	}
	if len(errs) != 0 {
		t.Fatalf("out-of-bounds price should be skipped silently; got errs=%v", errs)
	}
}

func TestCalculateAllCollectsPerOptionErrors(t *testing.T) {
	// Genuinely unexpected failures (here, an invalid zero strike) are collected as
	// per-option errors rather than skipped silently.
	engine := ivcalc.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 1000, StrikePrice: 0, ExpiryDate: "1405/12/15"},
	}
	got, errs := engine.CalculateAll(opts, 25000, 0.20)
	if len(got) != 0 {
		t.Fatalf("expected 0 results, got %d", len(got))
	}
	if len(errs) != 1 {
		t.Fatalf("len(errs)=%d want 1", len(errs))
	}
}
