package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/db"
)

func TestDatabaseConnection(t *testing.T) {
	if os.Getenv("SUPABASE_DB_HOST") == "" {
		t.Skip("SUPABASE_DB_HOST not set; skipping integration test")
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
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
