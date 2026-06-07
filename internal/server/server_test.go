package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"ahrm/internal/config"
	"ahrm/internal/server"
)

func TestHealthAlwaysOK(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, nil)
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

func TestReadyWithoutDatabase(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, nil)
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
	if report.Supabase.Configured {
		t.Fatal("expected supabase not configured")
	}
}

func TestReadyUsesReadinessReportFields(t *testing.T) {
	cfg := &config.Config{
		SourceArena: config.SourceArenaConfig{APIToken: "token"},
	}
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, nil)
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
