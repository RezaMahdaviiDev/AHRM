package arbitrage

import (
	"fmt"

	"ahrm/internal/pairs"
)

type Opportunity struct {
	Symbol         string  `json:"symbol"`
	SellRow1Volume float64 `json:"1_sell_volume"`
	BuyRow1Volume  float64 `json:"1_buy_volume"`
	Expiry         string  `json:"expiry"`
	Strike      float64 `json:"strike"`
	ReturnPct   float64 `json:"return_pct"`
	CallPrice   float64 `json:"call_price"`
	PutPrice    float64 `json:"put_price"`
	Underlying  float64 `json:"underlying"`
	Capital     float64 `json:"capital"`
	Spread      float64 `json:"spread"`
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
	return Opportunity{
		Symbol:         pair.Call.Name,
		SellRow1Volume: pair.Call.SellRow1Volume,
		BuyRow1Volume:  pair.Call.BuyRow1Volume,
		Expiry:         pair.ExpiryLabel,
		Strike:     pair.Strike,
		ReturnPct:  ret,
		CallPrice:  call,
		PutPrice:   put,
		Underlying: underlyingPrice,
		Capital:    capital,
		Spread:     spread,
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
