package matrix

import (
	"fmt"
	"sort"

	"ahrm/internal/domain"
	"ahrm/internal/sourcearena"
)

type Matrix struct {
	Kind    string    `json:"kind"`
	Symbols []string  `json:"symbols"`
	Prices  []float64 `json:"prices"`
	Cells   [][]float64 `json:"cells"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) BuildCalls(options []sourcearena.Option) (Matrix, error) {
	return e.build(options, domain.CallOptionPrefix, "call")
}

func (e *Engine) BuildPuts(options []sourcearena.Option) (Matrix, error) {
	return e.build(options, domain.PutOptionPrefix, "put")
}

func (e *Engine) build(options []sourcearena.Option, prefix, kind string) (Matrix, error) {
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
		return Matrix{Kind: kind}, fmt.Errorf("no %s options found", kind)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })

	symbols := make([]string, len(filtered))
	prices := make([]float64, len(filtered))
	for i, opt := range filtered {
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
	return Matrix{Kind: kind, Symbols: symbols, Prices: prices, Cells: cells}, nil
}
