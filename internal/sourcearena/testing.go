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

// NewTestClient builds a client pointed at a test server base URL.
func NewTestClient(cfg config.SourceArenaConfig, baseURL string, store RawStore) *Client {
	c := NewClient(cfg, store)
	c.v1Base = baseURL + "/"
	c.v2Base = baseURL
	return c
}
