package matrix

import (
	"fmt"
	"sort"

	"ahrm/internal/domain"
	"ahrm/internal/jalali"
	"ahrm/internal/sourcearena"
)

type Matrix struct {
	Kind    string      `json:"kind"`
	Expiry  string      `json:"expiry"`
	Symbols []string    `json:"symbols"`
	Prices  []float64   `json:"prices"`
	Cells   [][]float64 `json:"cells"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) BuildCalls(options []sourcearena.Option) ([]Matrix, error) {
	return e.build(options, domain.CallOptionPrefix, "call")
}

func (e *Engine) BuildPuts(options []sourcearena.Option) ([]Matrix, error) {
	return e.build(options, domain.PutOptionPrefix, "put")
}

func (e *Engine) build(options []sourcearena.Option, prefix, kind string) ([]Matrix, error) {
	filtered := make([]sourcearena.Option, 0)
	for _, opt := range options {
		if domain.IsCallOption(opt.Name) && prefix == domain.CallOptionPrefix {
			filtered = append(filtered, opt)
		}
		if domain.IsPutOption(opt.Name) && prefix == domain.PutOptionPrefix {
			filtered = append(filtered, opt)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no %s options found", kind)
	}

	groups := make(map[string][]sourcearena.Option)
	for _, opt := range filtered {
		groups[opt.ExpiryDate] = append(groups[opt.ExpiryDate], opt)
	}

	expiries := make([]string, 0, len(groups))
	for expiry := range groups {
		expiries = append(expiries, expiry)
	}
	sort.Slice(expiries, func(i, j int) bool {
		ti, ei := jalali.ParseDate(expiries[i])
		tj, ej := jalali.ParseDate(expiries[j])
		if ei != nil || ej != nil {
			return expiries[i] < expiries[j]
		}
		return ti.Before(tj)
	})

	matrices := make([]Matrix, 0, len(expiries))
	for _, expiry := range expiries {
		opts := groups[expiry]
		sort.Slice(opts, func(i, j int) bool { return opts[i].Name < opts[j].Name })

		symbols := make([]string, len(opts))
		prices := make([]float64, len(opts))
		for i, opt := range opts {
			symbols[i] = opt.Name
			prices[i] = opt.ClosePrice
		}
		cells := make([][]float64, len(prices))
		for i := range prices {
			cells[i] = make([]float64, len(prices))
			for j := range prices {
				cells[i][j] = prices[i] - prices[j]
			}
		}
		matrices = append(matrices, Matrix{
			Kind: kind, Expiry: expiry, Symbols: symbols, Prices: prices, Cells: cells,
		})
	}
	return matrices, nil
}
