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

	// K < S: StaticROI must show Max ROI formula instead of C/NetCost
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
	// K(12000) < S(25000): both static and max ROI use (K-NetCost)/NetCost formula
	wantMax := ((12000.0 - 23500.0) / 23500.0) * 100
	if diff := cc.StaticROIPct - wantMax; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("static_roi=%v want=%v (max formula, because K<S)", cc.StaticROIPct, wantMax)
	}
	if diff := cc.MaxROIPct - wantMax; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("max_roi=%v want=%v", cc.MaxROIPct, wantMax)
	}
	// NetCost = S - C (break-even, BreakEven field removed — verified via NetCost)
	wantBreakEven := 25000.0 - 1500.0
	if diff := cc.NetCost - wantBreakEven; diff > 0.01 || diff < -0.01 {
		t.Fatalf("net_cost=%v want=%v", cc.NetCost, wantBreakEven)
	}
}

func TestCalculateAllMaxROIWhenKGeS(t *testing.T) {
	engine := coveredcall.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

	// K >= S: MaxROI uses the normal formula ((K - NetCost) / NetCost) * 100
	// S=20000, C=1500, K=26000, NetCost=18500
	opts := []sourcearena.Option{
		{Name: "ضهرم2600", ClosePrice: 1500, StrikePrice: 26000, ExpiryDate: "1405/12/15"},
	}
	got, err := engine.CalculateAll(opts, 20000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	cc := got[0]
	wantMax := ((26000.0 - 18500.0) / 18500.0) * 100
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
