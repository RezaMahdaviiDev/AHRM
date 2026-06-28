package server_test

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/server"
)

func TestHealthAlwaysOK(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status field = %q", payload["status"])
	}
}

func TestReadyAlwaysOK(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var report config.Readiness
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if !report.ConfigLoaded {
		t.Fatal("expected config_loaded true")
	}
}

func TestEventsSSEDelivery(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, slog.Default(), nil)
	h := srv.Handler()

	ts := httptest.NewServer(h)
	defer ts.Close()

	// connect SSE client
	resp, err := http.Get(ts.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q", ct)
	}

	// publish from broadcaster
	done := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				done <- strings.TrimPrefix(line, "data: ")
				return
			}
		}
	}()

	time.Sleep(10 * time.Millisecond) // let client register
	srv.Broadcaster().Publish("test alert")

	select {
	case got := <-done:
		if got != "test alert" {
			t.Fatalf("got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}
}

func TestAlertsJSServed(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/static/alerts.js", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "EventSource") {
		t.Fatal("alerts.js must reference EventSource")
	}
}

func TestReadyUsesReadinessReportFields(t *testing.T) {
	cfg := &config.Config{
		SourceArena: config.SourceArenaConfig{APIToken: "token"},
	}
	srv := server.New(cfg, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	var report config.Readiness
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if !report.SourceArena.Configured {
		t.Fatal("expected sourcearena configured")
	}
}
