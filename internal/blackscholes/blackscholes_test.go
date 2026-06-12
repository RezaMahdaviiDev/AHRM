package blackscholes_test

import (
	"errors"
	"math"
	"testing"

	"ahrm/internal/blackscholes"
)

func TestCallPriceKnownValues(t *testing.T) {
	S, K, T, r, sigma := 100.0, 100.0, 1.0, 0.05, 0.2
	price := blackscholes.CallPrice(S, K, T, r, sigma)
	if price <= 0 {
		t.Fatalf("price=%v want positive", price)
	}
	// ATM 1Y call with 20% vol should be roughly 10
	if price < 8 || price > 14 {
		t.Fatalf("price=%v outside expected range [8,14]", price)
	}
}

func TestImpliedVolatilityRoundTrip(t *testing.T) {
	S, K, T, r, wantSigma := 25000.0, 12000.0, 180.0/365.0, 0.20, 0.35
	marketPrice := blackscholes.CallPrice(S, K, T, r, wantSigma)
	gotSigma, err := blackscholes.ImpliedVolatility(marketPrice, S, K, T, r)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(gotSigma-wantSigma) > 1e-3 {
		t.Fatalf("sigma=%v want=%v", gotSigma, wantSigma)
	}
}

func TestImpliedVolatilityOutOfBounds(t *testing.T) {
	S, K, T, r := 100.0, 100.0, 1.0, 0.05
	_, err := blackscholes.ImpliedVolatility(S+1, S, K, T, r)
	if !errors.Is(err, blackscholes.ErrPriceOutOfBounds) {
		t.Fatalf("err=%v want ErrPriceOutOfBounds", err)
	}
	_, err = blackscholes.ImpliedVolatility(-1, S, K, T, r)
	if !errors.Is(err, blackscholes.ErrPriceOutOfBounds) {
		t.Fatalf("err=%v want ErrPriceOutOfBounds", err)
	}
	_, err = blackscholes.ImpliedVolatility(10, S, K, 0, r)
	if !errors.Is(err, blackscholes.ErrInvalidInputs) {
		t.Fatalf("err=%v want ErrInvalidInputs", err)
	}
}
