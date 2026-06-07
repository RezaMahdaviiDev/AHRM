package sourcearena

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// flexFloat accepts JSON numbers or numeric strings (e.g. "24.56", "4%").
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	v, err := parseFlexibleFloat(data)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

type optionWire struct {
	Name                string    `json:"name"`
	ClosePrice          flexFloat `json:"close_price"`
	ClosePriceChangePct flexFloat `json:"close_price_change_percent"`
	StrikePrice         flexFloat `json:"emal_price"`
	BasisName           string    `json:"basis_name"`
	BasisPricePercent   flexFloat `json:"basis_price_percent"`
	OpenPosition        flexFloat `json:"op"`
	TradeValue          flexFloat `json:"trade_value"`
	LowestPrice         flexFloat `json:"lowest_price"`
	HighestPrice        flexFloat `json:"highest_price"`
	ExpiryDate          string    `json:"to_date"`
}

type symbolWire struct {
	Name                string    `json:"name"`
	ClosePrice          flexFloat `json:"close_price"`
	ClosePriceChangePct flexFloat `json:"close_price_change_percent"`
	TradeValue          flexFloat `json:"trade_value"`
}

type Option struct {
	Name                   string  `json:"name"`
	ClosePrice             float64 `json:"close_price"`
	ClosePriceChangePct    float64 `json:"close_price_change_percent"`
	StrikePrice            float64 `json:"emal_price"`
	ExpiryDate             string  `json:"to_date"`
	BasisName              string  `json:"basis_name"`
	BasisPricePercent      float64 `json:"basis_price_percent"`
	OpenPosition           float64 `json:"op"`
	TradeValue             float64 `json:"trade_value"`
	LowestPrice            float64 `json:"lowest_price"`
	HighestPrice           float64 `json:"highest_price"`
}

type SymbolQuote struct {
	Name                string  `json:"name"`
	ClosePrice          float64 `json:"close_price"`
	ClosePriceChangePct float64 `json:"close_price_change_percent"`
	TradeValue          float64 `json:"trade_value"`
}

type Candle struct {
	Close  float64 `json:"c"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Open   float64 `json:"o"`
	Volume float64 `json:"v"`
	Time   int64   `json:"t"`
}

func decodeOptions(raw json.RawMessage) ([]Option, error) {
	var wires []optionWire
	if err := json.Unmarshal(raw, &wires); err == nil {
		return normalizeOptions(wiresToOptions(wires)), nil
	}
	var list []Option
	if err := json.Unmarshal(raw, &list); err == nil {
		return normalizeOptions(list), nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("decode options: %w", err)
	}
	for _, key := range []string{"data", "results", "items"} {
		if payload, ok := wrapped[key]; ok {
			if err := json.Unmarshal(payload, &wires); err == nil {
				return normalizeOptions(wiresToOptions(wires)), nil
			}
			if err := json.Unmarshal(payload, &list); err == nil {
				return normalizeOptions(list), nil
			}
		}
	}
	if msg, ok := wrapped["Error"]; ok {
		return nil, NewAPIError("options", 0, strings.Trim(string(msg), `"`))
	}
	return nil, fmt.Errorf("decode options: unexpected payload")
}

func decodeSymbols(raw json.RawMessage) ([]SymbolQuote, error) {
	var wires []symbolWire
	if err := json.Unmarshal(raw, &wires); err == nil {
		return normalizeSymbols(wiresToSymbols(wires)), nil
	}
	var wire symbolWire
	if err := json.Unmarshal(raw, &wire); err == nil && wire.Name != "" {
		return normalizeSymbols(wiresToSymbols([]symbolWire{wire})), nil
	}
	var list []SymbolQuote
	if err := json.Unmarshal(raw, &list); err == nil {
		return normalizeSymbols(list), nil
	}
	var single SymbolQuote
	if err := json.Unmarshal(raw, &single); err == nil && single.Name != "" {
		return normalizeSymbols([]SymbolQuote{single}), nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("decode symbols: %w", err)
	}
	for _, key := range []string{"data", "results", "items"} {
		if payload, ok := wrapped[key]; ok {
			if err := json.Unmarshal(payload, &list); err == nil {
				return normalizeSymbols(list), nil
			}
		}
	}
	if msg, ok := wrapped["Error"]; ok {
		return nil, NewAPIError("symbols", 0, strings.Trim(string(msg), `"`))
	}
	return nil, fmt.Errorf("decode symbols: unexpected payload")
}

func decodeCandles(raw json.RawMessage) ([]Candle, error) {
	var list []Candle
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("decode candles: %w", err)
	}
	return list, nil
}

func wiresToOptions(wires []optionWire) []Option {
	out := make([]Option, 0, len(wires))
	for _, w := range wires {
		out = append(out, Option{
			Name:                w.Name,
			ClosePrice:          float64(w.ClosePrice),
			ClosePriceChangePct: float64(w.ClosePriceChangePct),
			StrikePrice:         float64(w.StrikePrice),
			ExpiryDate:          w.ExpiryDate,
			BasisName:           w.BasisName,
			BasisPricePercent:   float64(w.BasisPricePercent),
			OpenPosition:        float64(w.OpenPosition),
			TradeValue:          float64(w.TradeValue),
			LowestPrice:         float64(w.LowestPrice),
			HighestPrice:        float64(w.HighestPrice),
		})
	}
	return out
}

func wiresToSymbols(wires []symbolWire) []SymbolQuote {
	out := make([]SymbolQuote, 0, len(wires))
	for _, w := range wires {
		out = append(out, SymbolQuote{
			Name:                w.Name,
			ClosePrice:          float64(w.ClosePrice),
			ClosePriceChangePct: float64(w.ClosePriceChangePct),
			TradeValue:          float64(w.TradeValue),
		})
	}
	return out
}

func normalizeOptions(items []Option) []Option {
	out := make([]Option, 0, len(items))
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeSymbols(items []SymbolQuote) []SymbolQuote {
	out := make([]SymbolQuote, 0, len(items))
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func parseFlexibleFloat(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	return strconv.ParseFloat(s, 64)
}
