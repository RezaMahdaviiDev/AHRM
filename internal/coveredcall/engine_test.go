package coveredcall_test

import (
	"testing"
	"time"

	"ahrm/internal/coveredcall"
	"ahrm/internal/sourcearena"
)

func TestCalculateAllFormulas(t *testing.T) {
	engine := coveredcall.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 1500, StrikePrice: 12000, ExpiryDate: "1405/12/15"},
	}
	got, err := engine.CalculateAll(opts, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	cc := got[0]
	if cc.NetCost != 23500 {
		t.Fatalf("net_cost=%v", cc.NetCost)
	}
	wantStatic := (25000.0 / 23500.0) * 100
	if diff := cc.StaticROIPct - wantStatic; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("static_roi=%v want=%v", cc.StaticROIPct, wantStatic)
	}
	wantMax := ((12000.0 - 23500.0) / 23500.0) * 100
	if diff := cc.MaxROIPct - wantMax; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("max_roi=%v want=%v", cc.MaxROIPct, wantMax)
	}
}

func TestCalculateAllSkipsShortExpiry(t *testing.T) {
	engine := coveredcall.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1300", ClosePrice: 1200, StrikePrice: 13000, ExpiryDate: "1404/04/01"},
	}
	got, err := engine.CalculateAll(opts, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestCalculateAllSkipsNonPositiveNetCost(t *testing.T) {
	engine := coveredcall.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", ClosePrice: 26000, StrikePrice: 12000, ExpiryDate: "1405/12/15"},
	}
	got, err := engine.CalculateAll(opts, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestCalculateAllIgnoresPuts(t *testing.T) {
	engine := coveredcall.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "طهرم1200", ClosePrice: 800, StrikePrice: 12000, ExpiryDate: "1405/12/15"},
	}
	got, err := engine.CalculateAll(opts, 25000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}
