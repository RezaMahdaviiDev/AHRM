package pairs

import (
	"fmt"
	"time"

	"ahrm/internal/domain"
	"ahrm/internal/jalali"
	"ahrm/internal/sourcearena"
)

const MinDaysToExpiry = 30

type Pair struct {
	Call         sourcearena.Option
	Put          sourcearena.Option
	Strike       float64
	Expiry       time.Time
	ExpiryLabel  string
	DaysToExpiry int
}

type Engine struct {
	Now      func() time.Time
	MinDays  int
}

func NewEngine() *Engine {
	return &Engine{
		Now:     time.Now,
		MinDays: MinDaysToExpiry,
	}
}

func (e *Engine) Match(options []sourcearena.Option) ([]Pair, error) {
	if e.MinDays <= 0 {
		e.MinDays = MinDaysToExpiry
	}
	calls := filterCalls(options)
	puts := filterPuts(options)

	putIndex := make(map[string]sourcearena.Option)
	for _, put := range puts {
		key := pairKey(put.StrikePrice, put.ExpiryDate)
		putIndex[key] = put
	}

	now := e.Now().UTC()
	var out []Pair
	for _, call := range calls {
		put, ok := putIndex[pairKey(call.StrikePrice, call.ExpiryDate)]
		if !ok {
			continue
		}
		expiry, err := jalali.ParseDate(call.ExpiryDate)
		if err != nil {
			continue
		}
		days := jalali.CalendarDaysUntil(now, expiry)
		if days <= e.MinDays {
			continue
		}
		out = append(out, Pair{
			Call:         call,
			Put:          put,
			Strike:       call.StrikePrice,
			Expiry:       expiry,
			ExpiryLabel:  call.ExpiryDate,
			DaysToExpiry: days,
		})
	}
	return out, nil
}

func filterCalls(options []sourcearena.Option) []sourcearena.Option {
	var out []sourcearena.Option
	for _, opt := range options {
		if domain.IsCallOption(opt.Name) {
			out = append(out, opt)
		}
	}
	return out
}

func filterPuts(options []sourcearena.Option) []sourcearena.Option {
	var out []sourcearena.Option
	for _, opt := range options {
		if domain.IsPutOption(opt.Name) {
			out = append(out, opt)
		}
	}
	return out
}

func pairKey(strike float64, expiry string) string {
	return fmt.Sprintf("%v|%s", strike, expiry)
}
