package sourcearena

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ahrm/internal/config"
)

const (
	// Market data (options, symbols) — legacy REST API path structure.
	defaultMarketBase = "https://apis.sourcearena.ir/api/"
	// Daily/minute candles — v2 candle API.
	defaultCandleBase = "https://api3.sourcearena.ir/api/v2/candle/1m"

	// Price adjustment types for candle API (type query param).
	AdjustUnadjusted     = 0 // تعدیل نشده
	AdjustCapAndDividend = 1 // افزایش سرمایه و سود نقدی
	AdjustCapitalOnly    = 2 // افزایش سرمایه
	AdjustDividendOnly   = 3 // سود نقدی
	AdjustPerformance    = 4 // عملکردی

	Resolution1D = "1D"
)

// CandleRequest describes a SourceArena v2 candle fetch.
// Auth: X-Header-Token header (not query string).
type CandleRequest struct {
	Symbol     string
	From       time.Time
	To         time.Time
	Resolution string
	Type       int
}

func (r CandleRequest) withDefaults() CandleRequest {
	out := r
	if out.Resolution == "" {
		out.Resolution = Resolution1D
	}
	if out.Type < 0 {
		out.Type = AdjustCapAndDividend
	}
	return out
}

type Client struct {
	token      string
	marketBase string
	candleBase string
	httpClient *http.Client
	store      RawStore
}

type RawStore interface {
	SaveRaw(ctx context.Context, endpoint string, statusCode int, body []byte) error
}

func NewClient(cfg config.SourceArenaConfig, store RawStore) *Client {
	transport := &http.Transport{}
	if cfg.HTTPProxy != "" {
		if proxyURL, err := url.Parse(cfg.HTTPProxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &Client{
		token:      cfg.APIToken,
		marketBase: defaultMarketBase,
		candleBase: defaultCandleBase,
		httpClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: transport,
		},
		store: store,
	}
}

func (c *Client) FetchOptions(ctx context.Context) ([]Option, error) {
	raw, err := c.getMarket(ctx, c.marketURL("all=e"), "options")
	if err != nil {
		return nil, err
	}
	return decodeOptions(raw)
}

func (c *Client) FetchAllSymbols(ctx context.Context) ([]SymbolQuote, error) {
	// type=2 = all symbol types (bourse, OTC, funds, options, etc.) per SourceArena docs.
	raw, err := c.getMarket(ctx, c.marketURL("all&type=2"), "symbols_all")
	if err != nil {
		return nil, err
	}
	return decodeSymbols(raw)
}

func (c *Client) FetchClosedSymbols(ctx context.Context) ([]ClosedSymbol, error) {
	raw, err := c.getMarket(ctx, c.marketURL("closed_symbols"), "closed_symbols")
	if err != nil {
		return nil, err
	}
	return decodeClosedSymbols(raw)
}

func (c *Client) FetchSupervisorMessages(ctx context.Context) ([]SupervisorMessage, error) {
	raw, err := c.getMarket(ctx, c.marketURL("inspect=all"), "inspect")
	if err != nil {
		return nil, err
	}
	return decodeSupervisorMessages(raw)
}

func (c *Client) FetchSymbol(ctx context.Context, symbol string) (SymbolQuote, error) {
	raw, err := c.getMarket(ctx, c.marketURL("name="+url.QueryEscape(symbol)), "symbol")
	if err != nil {
		return SymbolQuote{}, err
	}
	items, err := decodeSymbols(raw)
	if err != nil {
		return SymbolQuote{}, err
	}
	if len(items) == 0 {
		return SymbolQuote{}, NewAPIError("symbol", 0, "empty response")
	}
	return items[0], nil
}

func (c *Client) FetchIndicators(ctx context.Context, symbol string) (*TechnicalIndicators, error) {
	raw, err := c.getMarket(ctx, c.marketURL("name="+url.QueryEscape(symbol)+"&all_indicators&adjusted=1"), "indicators")
	if err != nil {
		return nil, err
	}
	return decodeTechnicalIndicators(raw)
}

func (c *Client) FetchDailyCandles(ctx context.Context, symbol string, from, to time.Time) ([]Candle, error) {
	return c.FetchCandles(ctx, CandleRequest{
		Symbol: symbol,
		From:   from,
		To:     to,
		Type:   AdjustCapAndDividend,
	})
}

func (c *Client) FetchCandles(ctx context.Context, req CandleRequest) ([]Candle, error) {
	req = req.withDefaults()
	query := url.Values{}
	query.Set("from", fmt.Sprintf("%d", req.From.Unix()))
	query.Set("to", fmt.Sprintf("%d", req.To.Unix()))
	query.Set("symbol", req.Symbol)
	query.Set("resolution", req.Resolution)
	query.Set("type", fmt.Sprintf("%d", req.Type))
	endpoint := c.candleBase + "?" + query.Encode()
	raw, err := c.getCandle(ctx, endpoint, "candles")
	if err != nil {
		return nil, err
	}
	return decodeCandles(raw)
}

// marketURL builds options/symbols endpoints on apis.sourcearena.ir.
// Auth: token query param only (header auth is not supported on this host).
func (c *Client) marketURL(params string) string {
	endpoint := strings.TrimSuffix(c.marketBase, "/") + "/?token=" + url.QueryEscape(c.token)
	if params != "" {
		endpoint += "&" + params
	}
	return endpoint
}

func (c *Client) getMarket(ctx context.Context, endpoint, label string) (json.RawMessage, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, NewAPIError(label, 0, "SOURCEARENA_API_TOKEN is not configured")
	}
	return c.doGet(ctx, endpoint, label, false)
}

func (c *Client) getCandle(ctx context.Context, endpoint, label string) (json.RawMessage, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, NewAPIError(label, 0, "SOURCEARENA_API_TOKEN is not configured")
	}
	return c.doGet(ctx, endpoint, label, true)
}

func (c *Client) doGet(ctx context.Context, endpoint, label string, useHeaderAuth bool) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if useHeaderAuth {
		req.Header.Set("X-Header-Token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewAPIError(label, 0, err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.store != nil {
		_ = c.store.SaveRaw(ctx, label, resp.StatusCode, body)
	}
	if resp.StatusCode >= 400 {
		return nil, NewAPIError(label, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := checkAPIEnvelope(label, body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func checkAPIEnvelope(label string, body []byte) error {
	var env struct {
		Success   *bool  `json:"success"`
		Message   string `json:"message"`
		ErrorCode int    `json:"error_code"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Success == nil {
		return nil
	}
	if *env.Success {
		return nil
	}
	msg := strings.TrimSpace(env.Message)
	if msg == "" {
		msg = "request failed"
	}
	if env.ErrorCode != 0 {
		msg = fmt.Sprintf("%s (error_code=%d)", msg, env.ErrorCode)
	}
	return NewAPIError(label, 0, msg)
}
