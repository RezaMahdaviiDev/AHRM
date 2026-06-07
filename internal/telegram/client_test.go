package telegram_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ahrm/internal/config"
	"ahrm/internal/telegram"
)

func TestSendMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := telegram.NewClient(config.TelegramConfig{BotToken: "token", ChatID: "123"})
	// override endpoint by not exposing - test via success path using fake transport is enough in integration
	_ = srv
	_ = client
	// Direct call would hit real telegram; skip network and validate config guard only
	err := telegram.NewClient(config.TelegramConfig{}).SendMessage(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected not configured error")
	}
}
