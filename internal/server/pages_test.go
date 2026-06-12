package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"log/slog"

	"ahrm/internal/config"
	"ahrm/internal/scanner"
	"ahrm/internal/server"
)

func TestDashboardPageRenders(t *testing.T) {
	cfg := &config.Config{}
	scan := scanner.NewService(cfg, nil, nil, nil)
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, scan)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body, _ := io.ReadAll(rec.Body)
	text := string(body)
	if !strings.Contains(text, "AHRM") {
		t.Fatalf("unexpected body: %s", body)
	}
	if !strings.Contains(text, "آخرین به‌روزرسانی") || !strings.Contains(text, "(ایران)") {
		t.Fatalf("expected Iran update bar in body")
	}
}

func TestHealthStillIndependent(t *testing.T) {
	cfg := &config.Config{}
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestSnapshotCache(t *testing.T) {
	cfg := &config.Config{}
	scan := scanner.NewService(cfg, nil, nil, nil)
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, scan)
	h := srv.Handler()
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatal(rec.Code)
		}
	}
	_ = time.Second
}

func TestMatrixPageRenders(t *testing.T) {
	cfg := &config.Config{}
	scan := scanner.NewService(cfg, nil, nil, nil)
	srv := server.New(cfg, nil, slog.Default(), "migrations", false, scan)

	req := httptest.NewRequest(http.MethodGet, "/matrix", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body, _ := io.ReadAll(rec.Body)
	text := string(body)
	if !strings.Contains(text, "AHRM") {
		t.Fatalf("unexpected body: %s", body)
	}
}
