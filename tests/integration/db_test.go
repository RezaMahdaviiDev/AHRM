package integration_test

import (
	"context"
	"testing"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/db"
)

func TestDatabaseConnection(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Supabase.Configured() {
		t.Skip("supabase not fully configured; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()

	if err := db.Ping(ctx, pool); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}
