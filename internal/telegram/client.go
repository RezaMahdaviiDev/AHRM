package telegram

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
	token  string
	chatID string
	http   *http.Client
}

func NewClient(cfg config.TelegramConfig) *Client {
	return &Client{
		token:  cfg.BotToken,
		chatID: cfg.ChatID,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SendMessage(ctx context.Context, text string) error {
	if strings.TrimSpace(c.token) == "" || strings.TrimSpace(c.chatID) == "" {
		return fmt.Errorf("telegram is not configured")
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)
	form := url.Values{}
	form.Set("chat_id", c.chatID)
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
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		OK bool `json:"ok"`
	}
	_ = json.Unmarshal(body, &parsed)
	if !parsed.OK {
		return fmt.Errorf("telegram response not ok: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
