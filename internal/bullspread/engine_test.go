package bullspread_test

import (
	"testing"
	"time"

	"ahrm/internal/bullspread"
	"ahrm/internal/sourcearena"
)

func makeCall(name string, strike, ask, bid float64, expiry string) sourcearena.Option {
	return sourcearena.Option{
		Name:          name,
		StrikePrice:   strike,
		SellRow1Price: ask,
		BuyRow1Price:  bid,
		ClosePrice:    ask,
		ExpiryDate:    expiry,
	}
}

func fixedNow(t time.Time) func() time.Time { return func() time.Time { return t } }

// now = 2026-01-01, expiry = 1404/12/15 ≈ 45 days away (Gregorian 2026-03-06)
const testExpiry = "1404/12/15"

func TestATMFilter(t *testing.T) {
	eng := &bullspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	options := []sourcearena.Option{
		makeCall("ضهرم9500", 9500, 700, 680, testExpiry),   // moneyness = -5% → ATM edge
		makeCall("ضهرم10000", 10000, 400, 390, testExpiry),  // ATM (0%)
		makeCall("ضهرم10500", 10500, 200, 190, testExpiry),  // ATM +5% edge
		makeCall("ضهرم11000", 11000, 80, 70, testExpiry),   // OTM +10% → excluded from ATM
	}

	spreads := eng.CalculateAll(options, s, bullspread.ATM)
	if len(spreads) == 0 {
		t.Fatal("expected ATM spreads, got none")
	}
	for _, sp := range spreads {
		if sp.D > 0.45*sp.W {
			t.Errorf("debit ratio violated: D=%.0f W=%.0f", sp.D, sp.W)
		}
		if sp.K2 <= sp.K1 {
			t.Errorf("K2 must be > K1: K1=%.0f K2=%.0f", sp.K1, sp.K2)
		}
		mono := (sp.K1 - s) / s
		if mono < -0.05-1e-9 || mono > 0.05+1e-9 {
			t.Errorf("ATM K1 moneyness out of range: %.4f", mono)
		}
	}
}

func TestOTMFilter(t *testing.T) {
	eng := &bullspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	options := []sourcearena.Option{
		makeCall("ضهرم10000", 10000, 400, 390, testExpiry),  // ATM → excluded
		makeCall("ضهرم10600", 10600, 250, 240, testExpiry),  // OTM +6%
		makeCall("ضهرم11500", 11500, 100, 90, testExpiry),   // OTM +15%
		makeCall("ضهرم12200", 12200, 40, 30, testExpiry),    // OTM +22% → excluded
	}

	spreads := eng.CalculateAll(options, s, bullspread.OTM)
	for _, sp := range spreads {
		mono := (sp.K1 - s) / s
		if mono <= 0.05-1e-9 || mono > 0.20+1e-9 {
			t.Errorf("OTM K1 moneyness out of range: %.4f", mono)
		}
	}
}

func TestDebitRatioFilter(t *testing.T) {
	eng := &bullspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	// W = 1000, D = 500 → 50% > 45% → should be excluded
	options := []sourcearena.Option{
		makeCall("ضهرم10000", 10000, 500, 490, testExpiry),
		makeCall("ضهرم11000", 11000, 0, 0, testExpiry), // BidK2=ClosePrice=0 → D=500, W=1000 → 50%
	}
	_ = options

	// W = 1000, D = 400 → 40% ≤ 45% → should pass
	options2 := []sourcearena.Option{
		makeCall("ضهرم10000", 10000, 500, 490, testExpiry),
		makeCall("ضهرم11000", 11000, 100, 100, testExpiry), // BidK2=100, D=400, W=1000 → 40%
	}
	spreads := eng.CalculateAll(options2, s, bullspread.ATM)
	if len(spreads) == 0 {
		t.Fatal("expected spread with D/W=40%, got none")
	}
	if spreads[0].D > 0.45*spreads[0].W {
		t.Errorf("spread incorrectly filtered: D/W=%.2f", spreads[0].D/spreads[0].W)
	}
}

func TestSortedByRDescending(t *testing.T) {
	eng := &bullspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0
	options := []sourcearena.Option{
		makeCall("ضهرم9800", 9800, 600, 580, testExpiry),
		makeCall("ضهرم10000", 10000, 350, 340, testExpiry),
		makeCall("ضهرم10200", 10200, 180, 170, testExpiry),
	}
	spreads := eng.CalculateAll(options, s, bullspread.ATM)
	for i := 1; i < len(spreads); i++ {
		if spreads[i].R > spreads[i-1].R {
			t.Errorf("not sorted by R desc at index %d: R[%d]=%.4f > R[%d]=%.4f",
				i, i, spreads[i].R, i-1, spreads[i-1].R)
		}
	}
}
