package bearputspread

import (
	"math"
	"sort"
	"time"

	"ahrm/internal/domain"
	"ahrm/internal/jalali"
	"ahrm/internal/sourcearena"
)

const (
	minDays          = 30
	maxDays          = 60
	maxDebitRatioATM = 0.45
	maxDebitRatioOTM = 0.25
)

type FilterKind int

const (
	ATM FilterKind = iota
	OTM
)

type Spread struct {
	K1Symbol     string  `json:"k1_symbol"`
	K2Symbol     string  `json:"k2_symbol"`
	Expiry       string  `json:"expiry"`
	DaysToExpiry int     `json:"days_to_expiry"`
	S            float64 `json:"s"`
	K1           float64 `json:"k1"`
	K2           float64 `json:"k2"`
	AskK1        float64 `json:"ask_k1"`
	BidK2        float64 `json:"bid_k2"`
	W            float64 `json:"w"`
	D            float64 `json:"d"`
	DWPct        float64 `json:"dw_pct"`
	MP           float64 `json:"mp"`
	R            float64 `json:"r"`
}

type Engine struct {
	Now func() time.Time
}

func NewEngine() *Engine {
	return &Engine{Now: time.Now}
}

func askPrice(opt sourcearena.Option) float64 {
	if opt.SellRow1Price > 0 {
		return opt.SellRow1Price
	}
	return opt.ClosePrice
}

func bidPrice(opt sourcearena.Option) float64 {
	if opt.BuyRow1Price > 0 {
		return opt.BuyRow1Price
	}
	return opt.ClosePrice
}

func (e *Engine) CalculateAll(options []sourcearena.Option, underlyingPrice float64, kind FilterKind) []Spread {
	if underlyingPrice <= 0 {
		return nil
	}

	now := e.Now().UTC()

	type group struct {
		expiry       string
		daysToExpiry int
		puts         []sourcearena.Option
	}

	byExpiry := make(map[string]*group)
	for _, opt := range options {
		if !domain.IsPutOption(opt.Name) {
			continue
		}
		if opt.ExpiryDate == "" || opt.StrikePrice <= 0 {
			continue
		}
		expiry, err := jalali.ParseDate(opt.ExpiryDate)
		if err != nil {
			continue
		}
		days := jalali.CalendarDaysUntil(now, expiry)
		if days < minDays || days > maxDays {
			continue
		}
		if _, ok := byExpiry[opt.ExpiryDate]; !ok {
			byExpiry[opt.ExpiryDate] = &group{expiry: opt.ExpiryDate, daysToExpiry: days}
		}
		byExpiry[opt.ExpiryDate].puts = append(byExpiry[opt.ExpiryDate].puts, opt)
	}

	var out []Spread
	for _, g := range byExpiry {
		sort.Slice(g.puts, func(i, j int) bool {
			return g.puts[i].StrikePrice < g.puts[j].StrikePrice
		})

		for i, k1Opt := range g.puts {
			k1 := k1Opt.StrikePrice
			moneyness := (k1 - underlyingPrice) / underlyingPrice

			switch kind {
			case ATM:
				if math.Abs(moneyness) > 0.05 {
					continue
				}
			case OTM:
				if moneyness > -0.05 || moneyness < -0.20 {
					continue
				}
			}

			askK1 := askPrice(k1Opt)

			for j := 0; j < i; j++ {
				k2Opt := g.puts[j]
				k2 := k2Opt.StrikePrice
				bidK2 := bidPrice(k2Opt)

				w := k1 - k2
				d := askK1 - bidK2
				if d <= 0 || w <= 0 {
					continue
				}
				limit := maxDebitRatioATM
				if kind == OTM {
					limit = maxDebitRatioOTM
				}
				if d > limit*w {
					continue
				}

				mp := w - d
				out = append(out, Spread{
					K1Symbol:     k1Opt.Name,
					K2Symbol:     k2Opt.Name,
					Expiry:       g.expiry,
					DaysToExpiry: g.daysToExpiry,
					S:            underlyingPrice,
					K1:           k1,
					K2:           k2,
					AskK1:        askK1,
					BidK2:        bidK2,
					W:            w,
					D:            d,
					DWPct:        d / w * 100,
					MP:           mp,
					R:            mp / d,
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].R > out[j].R })
	return out
}
