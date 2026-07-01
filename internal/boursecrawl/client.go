package boursecrawl

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	htmlTagPattern  = regexp.MustCompile(`<[^>]+>`)
	spacePattern    = regexp.MustCompile(`\s+`)
	dateTimePattern = regexp.MustCompile(`([12][0-9]{3}[/-][01]?[0-9][/-][0-3]?[0-9](?:\s+[0-2]?[0-9]:[0-5][0-9])?)`)
)

type Notice struct {
	Reason      string
	PublishedAt string
	RawSnippet  string
}

type Client struct {
	urlTemplate string
	userAgent   string
	httpClient  *http.Client
}

func NewClient(urlTemplate, userAgent string) *Client {
	urlTemplate = strings.TrimSpace(urlTemplate)
	if urlTemplate == "" {
		return nil
	}
	if userAgent == "" {
		userAgent = "AHRM/1.0 (+symbol-halt-fallback-crawler)"
	}
	return &Client{
		urlTemplate: urlTemplate,
		userAgent:   userAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.urlTemplate != ""
}

func (c *Client) FetchLatestNotice(ctx context.Context, symbol string) (Notice, error) {
	if !c.Enabled() {
		return Notice{}, fmt.Errorf("bourse crawler not configured")
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return Notice{}, fmt.Errorf("symbol is required")
	}
	endpoint := strings.ReplaceAll(c.urlTemplate, "{symbol}", url.QueryEscape(symbol))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Notice{}, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Notice{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Notice{}, err
	}
	if resp.StatusCode >= 400 {
		return Notice{}, fmt.Errorf("bourse crawl status %d", resp.StatusCode)
	}

	text := normalizeText(stripTags(string(body)))
	if text == "" {
		return Notice{}, fmt.Errorf("empty bourse page")
	}
	reason := extractReason(text)
	if reason == "" {
		return Notice{}, fmt.Errorf("halt notice not found in crawl payload")
	}
	return Notice{
		Reason:      reason,
		PublishedAt: extractDateTime(text),
		RawSnippet:  reason,
	}, nil
}

func stripTags(raw string) string {
	unescaped := html.UnescapeString(raw)
	noTags := htmlTagPattern.ReplaceAllString(unescaped, " ")
	return normalizeDigits(noTags)
}

func normalizeText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.TrimSpace(spacePattern.ReplaceAllString(raw, " "))
}

func extractReason(text string) string {
	keywords := []string{"توقف", "متوقف", "تعلیق"}
	for _, keyword := range keywords {
		index := strings.Index(text, keyword)
		if index == -1 {
			continue
		}
		start := index - 70
		if start < 0 {
			start = 0
		}
		end := index + 140
		if end > len(text) {
			end = len(text)
		}
		return strings.TrimSpace(text[start:end])
	}
	return ""
}

func extractDateTime(text string) string {
	matches := dateTimePattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func normalizeDigits(s string) string {
	replacer := strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
		"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	)
	return replacer.Replace(s)
}
