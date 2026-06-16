package coveredcall

import (
	"fmt"
	"time"

	"ahrm/internal/domain"
	"ahrm/internal/jalali"
	"ahrm/internal/pairs"
	"ahrm/internal/sourcearena"
)

type CoveredCall struct {
	Symbol       string  `json:"symbol"`
	Expiry       string  `json:"expiry"`
	DaysToExpiry int     `json:"days_to_expiry"`
	Underlying   float64 `json:"underlying"`   // S
	OptionPrice  float64 `json:"option_price"` // C
	Strike       float64 `json:"strike"`       // K
	NetCost      float64 `json:"net_cost"`
	StaticROIPct float64 `json:"static_roi_pct"`
	MaxROIPct    float64 `json:"max_roi_pct"`
	BreakEven    float64 `json:"break_even"`
}

type Engine struct {
	Now     func() time.Time
	MinDays int // default 30, like pairs.MinDaysToExpiry
}

func NewEngine() *Engine {
	return &Engine{
		Now:     time.Now,
		MinDays: pairs.MinDaysToExpiry,
	}
}

func (e *Engine) CalculateAll(calls []sourcearena.Option, underlyingPrice float64) ([]CoveredCall, error) {
	if underlyingPrice <= 0 {
		return nil, fmt.Errorf("underlying price must be positive")
	}
	if e.MinDays <= 0 {
		e.MinDays = pairs.MinDaysToExpiry
	}
	now := e.Now().UTC()
	out := make([]CoveredCall, 0)
	for _, opt := range calls {
		if !domain.IsCallOption(opt.Name) {
			continue
		}
		expiry, err := jalali.ParseDate(opt.ExpiryDate)
		if err != nil {
			continue
		}
		days := jalali.CalendarDaysUntil(now, expiry)
		if days <= e.MinDays {
			continue
		}
		cc, ok := calculate(opt, underlyingPrice, days)
		if !ok {
			continue
		}
		out = append(out, cc)
	}
	return out, nil
}

func calculate(opt sourcearena.Option, underlyingPrice float64, days int) (CoveredCall, bool) {
	s := underlyingPrice
	c := opt.ClosePrice
	k := opt.StrikePrice
	netCost := s - c
	if netCost <= 0 {
		return CoveredCall{}, false
	}
	staticROI := (c / netCost) * 100
	maxROI := ((k - netCost) / netCost) * 100
	if k < s {
		maxROI = staticROI
	}
	return CoveredCall{
		Symbol:       opt.Name,
		Expiry:       opt.ExpiryDate,
		DaysToExpiry: days,
		Underlying:   s,
		OptionPrice:  c,
		Strike:       k,
		NetCost:      netCost,
		StaticROIPct: staticROI,
		MaxROIPct:    maxROI,
		BreakEven:    k * (1 - staticROI/100),
	}, true
}
