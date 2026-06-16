package bale

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

type Client struct {
	token   string
	chatIDs []string
	http    *http.Client
}

func NewClient(cfg config.BaleConfig) *Client {
	var ids []string
	for _, id := range strings.Split(cfg.ChatIDs, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return &Client{
		token:   cfg.BotToken,
		chatIDs: ids,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SendMessage(ctx context.Context, text string) error {
	if strings.TrimSpace(c.token) == "" || len(c.chatIDs) == 0 {
		return fmt.Errorf("bale is not configured")
	}
	var errs []string
	for _, chatID := range c.chatIDs {
		if err := c.sendTo(ctx, chatID, text); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", chatID, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("bale send errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Client) sendTo(ctx context.Context, chatID, text string) error {
	endpoint := fmt.Sprintf("https://tapi.bale.ai/bot%s/sendMessage", c.token)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		OK bool `json:"ok"`
	}
	_ = json.Unmarshal(body, &parsed)
	if !parsed.OK {
		return fmt.Errorf("response not ok: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
