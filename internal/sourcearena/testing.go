package sourcearena

import (
	"encoding/json"

	"ahrm/internal/config"
)

func DecodeOptionsForTest(raw []byte) ([]Option, error) {
	return decodeOptions(json.RawMessage(raw))
}

func DecodeSymbolsForTest(raw json.RawMessage) ([]SymbolQuote, error) {
	return decodeSymbols(raw)
}

func DecodeCandlesForTest(raw []byte) ([]Candle, error) {
	return decodeCandles(json.RawMessage(raw))
}

// NewTestClient builds a client pointed at test server URLs.
// marketBaseURL serves options/symbols; candleBaseURL serves candles.
func NewTestClient(cfg config.SourceArenaConfig, marketBaseURL, candleBaseURL string, store RawStore) *Client {
	c := NewClient(cfg, store)
	c.marketBase = marketBaseURL + "/"
	c.candleBase = candleBaseURL
	return c
}
