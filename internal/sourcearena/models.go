package sourcearena

import (
	"encoding/json"
	"fmt"
	"sort"
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
	SellRow1Price       flexFloat `json:"1_sell_price"`
	SellRow1Volume      flexFloat `json:"1_sell_volume"`
	BuyRow1Price        flexFloat `json:"1_buy_price"`
	BuyRow1Volume       flexFloat `json:"1_buy_volume"`
	TradeValue          flexFloat `json:"trade_value"`
	TradeVolume         flexFloat `json:"trade_volume"`
	LowestPrice         flexFloat `json:"lowest_price"`
	HighestPrice        flexFloat `json:"highest_price"`
	ExpiryDate          string    `json:"to_date"`
}

type symbolWire struct {
	Name                string    `json:"name"`
	Market              string    `json:"market"`
	ClosePrice          flexFloat `json:"close_price"`
	ClosePriceChangePct flexFloat `json:"close_price_change_percent"`
	FinalPrice          flexFloat `json:"final_price"`
	FinalPriceChangePct flexFloat `json:"final_price_change_percent"`
	TradeValue          flexFloat `json:"trade_value"`
	BuyRow1Price        flexFloat `json:"1_buy_price"`
	BuyRow1Volume       flexFloat `json:"1_buy_volume"`
	SellRow1Price       flexFloat `json:"1_sell_price"`
	SellRow1Volume      flexFloat `json:"1_sell_volume"`
	HighestPrice        flexFloat `json:"highest_price"`
	LowestPrice         flexFloat `json:"lowest_price"`
}

type Option struct {
	Name                string  `json:"name"`
	ClosePrice          float64 `json:"close_price"`
	ClosePriceChangePct float64 `json:"close_price_change_percent"`
	StrikePrice         float64 `json:"emal_price"`
	ExpiryDate          string  `json:"to_date"`
	BasisName           string  `json:"basis_name"`
	BasisPricePercent   float64 `json:"basis_price_percent"`
	OpenPosition        float64 `json:"op"`
	SellRow1Price       float64 `json:"1_sell_price"`
	SellRow1Volume      float64 `json:"1_sell_volume"`
	BuyRow1Price        float64 `json:"1_buy_price"`
	BuyRow1Volume       float64 `json:"1_buy_volume"`
	TradeValue          float64 `json:"trade_value"`
	TradeVolume         float64 `json:"trade_volume"`
	LowestPrice         float64 `json:"lowest_price"`
	HighestPrice        float64 `json:"highest_price"`
}

type SymbolQuote struct {
	Name                string  `json:"name"`
	Market              string  `json:"market"`
	ClosePrice          float64 `json:"close_price"`
	ClosePriceChangePct float64 `json:"close_price_change_percent"`
	FinalPrice          float64 `json:"final_price"`
	FinalPriceChangePct float64 `json:"final_price_change_percent"`
	TradeValue          float64 `json:"trade_value"`
	BuyRow1Price        float64 `json:"1_buy_price"`
	BuyRow1Volume       float64 `json:"1_buy_volume"`
	SellRow1Price       float64 `json:"1_sell_price"`
	SellRow1Volume      float64 `json:"1_sell_volume"`
	HighestPrice        float64 `json:"highest_price"`
	LowestPrice         float64 `json:"lowest_price"`
}

type ClosedSymbol struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	HaltedAt string `json:"halted_at"`
	Message  string `json:"message"`
}

type SupervisorMessage struct {
	Symbol      string `json:"symbol"`
	Message     string `json:"message"`
	PublishedAt string `json:"published_at"`
}

type Candle struct {
	Close  float64 `json:"c"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Open   float64 `json:"o"`
	Volume float64 `json:"v"`
	Time   int64   `json:"timestamp"`
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
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("decode candles: %w", err)
	}
	for _, key := range []string{"data", "results", "items", "candles"} {
		if payload, ok := wrapped[key]; ok {
			if err := json.Unmarshal(payload, &list); err == nil {
				return list, nil
			}
		}
	}
	if msg, ok := wrapped["message"]; ok {
		return nil, NewAPIError("candles", 0, strings.Trim(string(msg), `"`))
	}
	return nil, fmt.Errorf("decode candles: unexpected payload")
}

func decodeClosedSymbols(raw json.RawMessage) ([]ClosedSymbol, error) {
	objects, err := decodeObjectList(raw)
	if err != nil {
		return nil, fmt.Errorf("decode closed symbols: %w", err)
	}
	out := make([]ClosedSymbol, 0, len(objects))
	for _, item := range objects {
		name := stringField(item, "name", "symbol", "namad", "symbol_name", "ticker")
		if name == "" {
			continue
		}
		out = append(out, ClosedSymbol{
			Name:     name,
			Status:   stringField(item, "status", "state", "symbol_status"),
			HaltedAt: stringField(item, "halt_time", "stopped_at", "stop_time", "time", "date", "datetime", "updated_at"),
			Message:  stringField(item, "message", "reason", "desc", "description", "title", "text"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func decodeSupervisorMessages(raw json.RawMessage) ([]SupervisorMessage, error) {
	objects, err := decodeObjectList(raw)
	if err != nil {
		return nil, fmt.Errorf("decode supervisor messages: %w", err)
	}
	out := make([]SupervisorMessage, 0, len(objects))
	for _, item := range objects {
		symbol := stringField(item, "symbol", "name", "namad", "symbol_name", "ticker")
		message := stringField(item, "message", "msg", "text", "desc", "description", "body")
		title := stringField(item, "title", "subject", "header")
		if message == "" {
			message = title
		} else if title != "" && !strings.Contains(message, title) {
			message = title + " — " + message
		}
		if symbol == "" || message == "" {
			continue
		}
		out = append(out, SupervisorMessage{
			Symbol:      symbol,
			Message:     message,
			PublishedAt: stringField(item, "published_at", "publish_at", "created_at", "date", "datetime", "time", "updated_at"),
		})
	}
	return out, nil
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
			SellRow1Price:       float64(w.SellRow1Price),
			SellRow1Volume:      float64(w.SellRow1Volume),
			BuyRow1Price:        float64(w.BuyRow1Price),
			BuyRow1Volume:       float64(w.BuyRow1Volume),
			TradeValue:          float64(w.TradeValue),
			TradeVolume:         float64(w.TradeVolume),
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
			Market:              w.Market,
			ClosePrice:          float64(w.ClosePrice),
			ClosePriceChangePct: float64(w.ClosePriceChangePct),
			FinalPrice:          float64(w.FinalPrice),
			FinalPriceChangePct: float64(w.FinalPriceChangePct),
			TradeValue:          float64(w.TradeValue),
			BuyRow1Price:        float64(w.BuyRow1Price),
			BuyRow1Volume:       float64(w.BuyRow1Volume),
			SellRow1Price:       float64(w.SellRow1Price),
			SellRow1Volume:      float64(w.SellRow1Volume),
			HighestPrice:        float64(w.HighestPrice),
			LowestPrice:         float64(w.LowestPrice),
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

func decodeObjectList(raw json.RawMessage) ([]map[string]any, error) {
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("unexpected payload: %w", err)
	}
	if msg, ok := wrapped["Error"]; ok {
		return nil, NewAPIError("sourcearena", 0, strings.Trim(string(msg), `"`))
	}
	for _, key := range []string{"data", "results", "items", "list"} {
		payload, ok := wrapped[key]
		if !ok {
			continue
		}
		if err := json.Unmarshal(payload, &list); err == nil {
			return list, nil
		}
		var single map[string]any
		if err := json.Unmarshal(payload, &single); err == nil {
			if nested := flattenObjectValues(single); len(nested) > 0 {
				return nested, nil
			}
			return []map[string]any{single}, nil
		}
	}
	var single map[string]any
	if err := json.Unmarshal(raw, &single); err == nil {
		if nested := flattenObjectValues(single); len(nested) > 0 {
			return nested, nil
		}
		return []map[string]any{single}, nil
	}
	return nil, fmt.Errorf("unexpected payload")
}

func flattenObjectValues(item map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(item))
	for key, raw := range item {
		child, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := child["name"]; !exists {
			child["name"] = key
		}
		out = append(out, child)
	}
	if len(out) == len(item) && len(out) > 0 {
		return out
	}
	return nil
}

func stringField(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			if txt := strings.TrimSpace(toString(value)); txt != "" {
				return txt
			}
		}
	}
	return ""
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch value := v.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case json.Number:
		return value.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

// TechnicalIndicators is the response from the all_indicators endpoint.
// Field names are based on SourceArena API docs; may need adjustment after live testing.
type IndicatorItem struct {
	Value  float64 `json:"value"`
	Signal string  `json:"signal"` // "buy", "neutral", "sell"
}

type MACDItem struct {
	Value      float64 `json:"value"`
	SignalLine float64 `json:"signal_line"`
	Histogram  float64 `json:"histogram"`
	Signal     string  `json:"signal"`
}

type BollingerItem struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
	Signal string  `json:"signal"`
}

type TechnicalSum struct {
	Buy     int `json:"buy"`
	Neutral int `json:"neutral"`
	Sell    int `json:"sell"`
}

type TechnicalIndicators struct {
	TechnicalSum TechnicalSum  `json:"technical_sum"`
	RSI          IndicatorItem `json:"rsi"`
	MFI          IndicatorItem `json:"mfi"`
	CCI          IndicatorItem `json:"cci"`
	MACD         MACDItem      `json:"macd"`
	EMA9         IndicatorItem `json:"ema9"`
	EMA26        IndicatorItem `json:"ema26"`
	EMA50        IndicatorItem `json:"ema50"`
	SMA          IndicatorItem `json:"sma"`
	Bollinger    BollingerItem `json:"bollinger"`
}

func decodeTechnicalIndicators(raw json.RawMessage) (*TechnicalIndicators, error) {
	var ind TechnicalIndicators
	if err := json.Unmarshal(raw, &ind); err == nil {
		return &ind, nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("decode indicators: %w", err)
	}
	for _, key := range []string{"data", "results", "items"} {
		if payload, ok := wrapped[key]; ok {
			if err := json.Unmarshal(payload, &ind); err == nil {
				return &ind, nil
			}
		}
	}
	if msg, ok := wrapped["Error"]; ok {
		return nil, NewAPIError("indicators", 0, strings.Trim(string(msg), `"`))
	}
	return nil, fmt.Errorf("decode indicators: unexpected payload")
}
