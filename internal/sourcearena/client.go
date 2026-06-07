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
	defaultV1Base = "https://apis.sourcearena.ir/api/"
	defaultV2Base = "https://api3.sourcearena.ir/api/v2/candle/1m"
)

type Client struct {
	token      string
	v1Base     string
	v2Base     string
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
		token:  cfg.APIToken,
		v1Base: defaultV1Base,
		v2Base: defaultV2Base,
		httpClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: transport,
		},
		store: store,
	}
}

func (c *Client) FetchOptions(ctx context.Context) ([]Option, error) {
	raw, err := c.get(ctx, c.v1URL("all=e"), "options")
	if err != nil {
		return nil, err
	}
	return decodeOptions(raw)
}

func (c *Client) FetchAllSymbols(ctx context.Context) ([]SymbolQuote, error) {
	raw, err := c.get(ctx, c.v1URL("all"), "symbols_all")
	if err != nil {
		return nil, err
	}
	return decodeSymbols(raw)
}

func (c *Client) FetchSymbol(ctx context.Context, symbol string) (SymbolQuote, error) {
	raw, err := c.get(ctx, c.v1URL("name="+url.QueryEscape(symbol)), "symbol")
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

func (c *Client) FetchDailyCandles(ctx context.Context, symbol string, from, to time.Time) ([]Candle, error) {
	query := url.Values{}
	query.Set("from", fmt.Sprintf("%d", from.Unix()))
	query.Set("to", fmt.Sprintf("%d", to.Unix()))
	query.Set("symbol", symbol)
	query.Set("resolution", "1D")
	query.Set("type", "1")
	endpoint := c.v2Base + "?" + query.Encode()
	raw, err := c.get(ctx, endpoint, "candles")
	if err != nil {
		return nil, err
	}
	return decodeCandles(raw)
}

func (c *Client) v1URL(params string) string {
	endpoint := strings.TrimSuffix(c.v1Base, "/") + "/?token=" + url.QueryEscape(c.token)
	if params != "" {
		endpoint += "&" + params
	}
	return endpoint
}

func (c *Client) get(ctx context.Context, endpoint, label string) (json.RawMessage, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, NewAPIError(label, 0, "SOURCEARENA_API_TOKEN is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Header-Token", c.token)

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
	return json.RawMessage(body), nil
}
