package ivcalc

import (
	"fmt"
	"time"

	"ahrm/internal/blackscholes"
	"ahrm/internal/domain"
	"ahrm/internal/jalali"
	"ahrm/internal/pairs"
	"ahrm/internal/sourcearena"
)

type IVResult struct {
	Symbol       string  `json:"symbol"`
	Strike       float64 `json:"strike"`
	Expiry       string  `json:"expiry"`
	DaysToExpiry int     `json:"days_to_expiry"`
	OptionPrice  float64 `json:"option_price"`
	Underlying   float64 `json:"underlying"`
	IVPct        float64 `json:"iv_pct"`
}

type Engine struct {
	Now     func() time.Time
	MinDays int
}

func NewEngine() *Engine {
	return &Engine{
		Now:     time.Now,
		MinDays: pairs.MinDaysToExpiry,
	}
}

func (e *Engine) CalculateAll(options []sourcearena.Option, underlyingPrice, riskFreeRate float64) ([]IVResult, []string) {
	if underlyingPrice <= 0 {
		return nil, []string{"iv: underlying price must be positive"}
	}
	if e.MinDays <= 0 {
		e.MinDays = pairs.MinDaysToExpiry
	}
	now := e.Now().UTC()
	out := make([]IVResult, 0)
	errs := make([]string, 0)
	for _, opt := range options {
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
		T := float64(days) / 365.0
		sigma, err := blackscholes.ImpliedVolatility(opt.ClosePrice, underlyingPrice, opt.StrikePrice, T, riskFreeRate)
		if err != nil {
			if err != blackscholes.ErrPriceOutOfBounds {
				errs = append(errs, fmt.Sprintf("iv %s: %v", opt.Name, err))
			}
			continue
		}
		out = append(out, IVResult{
			Symbol:       opt.Name,
			Strike:       opt.StrikePrice,
			Expiry:       opt.ExpiryDate,
			DaysToExpiry: days,
			OptionPrice:  opt.ClosePrice,
			Underlying:   underlyingPrice,
			IVPct:        sigma * 100,
		})
	}
	return out, errs
}
