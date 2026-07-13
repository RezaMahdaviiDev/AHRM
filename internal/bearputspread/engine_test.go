package bearputspread_test

import (
	"testing"
	"time"

	"ahrm/internal/bearputspread"
	"ahrm/internal/sourcearena"
)

func makePut(name string, strike, ask, bid float64, expiry string) sourcearena.Option {
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

const testExpiry = "1404/12/15"

func TestATMFilter(t *testing.T) {
	eng := &bearputspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	options := []sourcearena.Option{
		makePut("طهرم9500", 9500, 700, 680, testExpiry),
		makePut("طهرم10000", 10000, 400, 390, testExpiry),
		makePut("طهرم10500", 10500, 200, 190, testExpiry),
		makePut("طهرم8000", 8000, 80, 70, testExpiry),
	}

	spreads := eng.CalculateAll(options, s, bearputspread.ATM)
	if len(spreads) == 0 {
		t.Fatal("expected ATM spreads, got none")
	}
	for _, sp := range spreads {
		if sp.D > 0.45*sp.W {
			t.Errorf("debit ratio violated: D=%.0f W=%.0f", sp.D, sp.W)
		}
		if sp.K1 <= sp.K2 {
			t.Errorf("K1 must be > K2: K1=%.0f K2=%.0f", sp.K1, sp.K2)
		}
		mono := (sp.K1 - s) / s
		if mono < -0.05-1e-9 || mono > 0.05+1e-9 {
			t.Errorf("ATM K1 moneyness out of range: %.4f", mono)
		}
	}
}

func TestOTMFilter(t *testing.T) {
	eng := &bearputspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	options := []sourcearena.Option{
		makePut("طهرم10000", 10000, 400, 390, testExpiry),
		makePut("طهرم9400", 9400, 250, 240, testExpiry),
		makePut("طهرم8500", 8500, 100, 90, testExpiry),
		makePut("طهرم7800", 7800, 40, 30, testExpiry),
	}

	spreads := eng.CalculateAll(options, s, bearputspread.OTM)
	for _, sp := range spreads {
		mono := (sp.K1 - s) / s
		if mono > -0.05+1e-9 || mono < -0.20-1e-9 {
			t.Errorf("OTM K1 moneyness out of range: %.4f", mono)
		}
	}
}

func TestDebitRatioFilter(t *testing.T) {
	eng := &bearputspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0

	options := []sourcearena.Option{
		makePut("طهرم10000", 10000, 500, 490, testExpiry),
		makePut("طهرم9000", 9000, 100, 100, testExpiry),
	}
	spreads := eng.CalculateAll(options, s, bearputspread.ATM)
	if len(spreads) == 0 {
		t.Fatal("expected spread with D/W=40%, got none")
	}
	if spreads[0].D > 0.45*spreads[0].W {
		t.Errorf("spread incorrectly filtered: D/W=%.2f", spreads[0].D/spreads[0].W)
	}
}

func TestSortedByRDescending(t *testing.T) {
	eng := &bearputspread.Engine{Now: fixedNow(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))}
	s := 10000.0
	options := []sourcearena.Option{
		makePut("طهرم10200", 10200, 600, 580, testExpiry),
		makePut("طهرم10000", 10000, 350, 340, testExpiry),
		makePut("طهرم9800", 9800, 180, 170, testExpiry),
	}
	spreads := eng.CalculateAll(options, s, bearputspread.ATM)
	for i := 1; i < len(spreads); i++ {
		if spreads[i].R > spreads[i-1].R {
			t.Errorf("not sorted by R desc at index %d", i)
		}
	}
}
