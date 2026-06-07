package sourcearena

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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
