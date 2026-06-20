package blackscholes

import (
	"errors"
	"math"
)

const (
	minSigma       = 0.001
	maxSigma       = 5.0
	maxIterations  = 100
	priceTolerance = 0.01
)

var (
	ErrInvalidInputs   = errors.New("invalid inputs: S, K, and T must be positive")
	ErrPriceOutOfBounds = errors.New("market price outside theoretical bounds")
	ErrNoConvergence   = errors.New("implied volatility did not converge")
)

func normCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func CallPrice(S, K, T, r, sigma float64) float64 {
	if T <= 0 || sigma <= 0 || S <= 0 || K <= 0 {
		return 0
	}
	sqrtT := math.Sqrt(T)
	d1 := (math.Log(S/K) + (r+0.5*sigma*sigma)*T) / (sigma * sqrtT)
	d2 := d1 - sigma*sqrtT
	return S*normCDF(d1) - K*math.Exp(-r*T)*normCDF(d2)
}

func theoreticalBounds(S, K, T, r float64) (lower, upper float64) {
	upper = S
	lower = S - K*math.Exp(-r*T)
	if lower < 0 {
		lower = 0
	}
	return lower, upper
}

func ImpliedVolatility(marketPrice, S, K, T, r float64) (float64, error) {
	if T <= 0 || S <= 0 || K <= 0 {
		return 0, ErrInvalidInputs
	}
	lower, upper := theoreticalBounds(S, K, T, r)
	// Allow a small tolerance for stale prices on illiquid deep-ITM options.
	const boundsTolerance = 0.5
	if marketPrice < lower-boundsTolerance || marketPrice >= upper {
		return 0, ErrPriceOutOfBounds
	}

	lo, hi := minSigma, maxSigma
	for i := 0; i < maxIterations; i++ {
		mid := (lo + hi) / 2
		price := CallPrice(S, K, T, r, mid)
		diff := price - marketPrice
		if math.Abs(diff) < priceTolerance {
			return mid, nil
		}
		if diff > 0 {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2, ErrNoConvergence
}
