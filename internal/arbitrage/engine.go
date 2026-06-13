package arbitrage

import (
	"fmt"

	"ahrm/internal/pairs"
)

type Opportunity struct {
	Symbol      string  `json:"symbol"`
	TradeVolume float64 `json:"trade_volume"`
	Expiry      string  `json:"expiry"`
	Strike      float64 `json:"strike"`
	ReturnPct   float64 `json:"return_pct"`
	CallPrice   float64 `json:"call_price"`
	PutPrice    float64 `json:"put_price"`
	Underlying  float64 `json:"underlying"`
	Capital     float64 `json:"capital"`
	Spread      float64 `json:"spread"`
	ReturnPct12_5 float64 `json:"return_pct_12_5"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) Calculate(pair pairs.Pair, underlyingPrice float64) (Opportunity, error) {
	if underlyingPrice <= 0 {
		return Opportunity{}, fmt.Errorf("underlying price must be positive")
	}
	if pair.Strike <= 0 {
		return Opportunity{}, fmt.Errorf("strike must be positive")
	}
	call := pair.Call.ClosePrice
	put := pair.Put.ClosePrice
	spread := call - put
	capital := underlyingPrice - spread
	if capital <= 0 {
		return Opportunity{}, fmt.Errorf("capital must be positive")
	}
	ret := ((pair.Strike - capital) / capital) * 100

	s12_5 := underlyingPrice * 1.125
	capital12_5 := s12_5 - spread
	var ret12_5 float64
	if capital12_5 != 0 {
		ret12_5 = ((pair.Strike - capital12_5) / capital12_5) * 100
	}

	return Opportunity{
		Symbol:        pair.Call.Name,
		TradeVolume:   pair.Call.TradeVolume,
		Expiry:        pair.ExpiryLabel,
		Strike:        pair.Strike,
		ReturnPct:     ret,
		CallPrice:     call,
		PutPrice:      put,
		Underlying:    underlyingPrice,
		Capital:       capital,
		Spread:        spread,
		ReturnPct12_5: ret12_5,
	}, nil
}

func (e *Engine) CalculateAll(pairs []pairs.Pair, underlyingPrice float64) ([]Opportunity, error) {
	out := make([]Opportunity, 0, len(pairs))
	for _, pair := range pairs {
		opp, err := e.Calculate(pair, underlyingPrice)
		if err != nil {
			continue
		}
		out = append(out, opp)
	}
	return out, nil
}
