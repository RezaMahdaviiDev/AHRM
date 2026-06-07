package pairs_test

import (
	"testing"
	"time"

	"ahrm/internal/pairs"
	"ahrm/internal/sourcearena"
)

func TestMatchPairsSameStrikeExpiry(t *testing.T) {
	engine := pairs.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1200", StrikePrice: 12000, ExpiryDate: "1405/12/15", ClosePrice: 1500},
		{Name: "طهرم1200", StrikePrice: 12000, ExpiryDate: "1405/12/15", ClosePrice: 800},
	}
	got, err := engine.Match(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len(pairs)=%d want 1", len(got))
	}
	if got[0].Strike != 12000 {
		t.Fatalf("strike=%v", got[0].Strike)
	}
}

func TestMatchPairsExpiryFilter(t *testing.T) {
	engine := pairs.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضهرم1300", StrikePrice: 13000, ExpiryDate: "1404/04/01", ClosePrice: 1200},
		{Name: "طهرم1300", StrikePrice: 13000, ExpiryDate: "1404/04/01", ClosePrice: 700},
	}
	got, err := engine.Match(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(got))
	}
}

func TestMatchPairsIgnoresOtherSymbols(t *testing.T) {
	engine := pairs.NewEngine()
	engine.Now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	opts := []sourcearena.Option{
		{Name: "ضفمل1200", StrikePrice: 12000, ExpiryDate: "1404/09/15", ClosePrice: 1500},
		{Name: "طهرم1200", StrikePrice: 12000, ExpiryDate: "1404/09/15", ClosePrice: 800},
	}
	got, err := engine.Match(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(got))
	}
}
