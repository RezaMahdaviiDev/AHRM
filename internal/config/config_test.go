package config_test

import (
	"testing"

	"ahrm/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SUPABASE_DB_HOST", "")
	t.Setenv("SUPABASE_DB_PORT", "")
	t.Setenv("SUPABASE_DB_NAME", "")
	t.Setenv("SUPABASE_DB_USER", "")
	t.Setenv("SUPABASE_DB_PASSWORD", "")
	t.Setenv("SUPABASE_DB_SSLMODE", "")
	t.Setenv("SOURCEARENA_API_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("RISK_FREE_RATE", "")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.RiskFreeRate != 0.20 {
		t.Fatalf("RiskFreeRate = %v, want 0.20", cfg.RiskFreeRate)
	}
	if cfg.SnapshotRefreshSeconds != 180 {
		t.Fatalf("SnapshotRefreshSeconds = %v, want 180", cfg.SnapshotRefreshSeconds)
	}
}

func TestValidatePartialSupabaseFails(t *testing.T) {
	cfg := &config.Config{
		Supabase: config.SupabaseConfig{
			Enabled:  true,
			Host:     "db.example.com",
			Password: "",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for partial supabase config")
	}
}

func TestValidateSupabaseDisabledIgnoresPartialFields(t *testing.T) {
	cfg := &config.Config{
		Supabase: config.SupabaseConfig{
			Enabled: false,
			Host:    "db.example.com",
			Port:    "5432",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Supabase.Configured() {
		t.Fatal("expected supabase not configured when disabled")
	}
}

func TestValidateCompleteSupabaseOK(t *testing.T) {
	cfg := &config.Config{
		Supabase: config.SupabaseConfig{
			Enabled:  true,
			Host:     "db.example.com",
			Port:     "5432",
			Name:     "postgres",
			User:     "postgres",
			Password: "secret",
			SSLMode:  "require",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSupabaseDSNEncodesPassword(t *testing.T) {
	cfg := &config.Config{
		Supabase: config.SupabaseConfig{
			Enabled:  true,
			Host:     "db.example.com",
			Port:     "5432",
			Name:     "postgres",
			User:     "postgres",
			Password: "p@ss,wrd",
			SSLMode:  "require",
		},
	}
	dsn, err := cfg.SupabaseDSN()
	if err != nil {
		t.Fatalf("SupabaseDSN() error = %v", err)
	}
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
}

func TestReadinessReportUnconfigured(t *testing.T) {
	cfg := &config.Config{}
	report := cfg.ReadinessReport(false)
	if !report.ConfigLoaded {
		t.Fatal("expected config_loaded true")
	}
	if report.Supabase.Configured {
		t.Fatal("expected supabase not configured")
	}
	if report.SourceArena.Configured {
		t.Fatal("expected sourcearena not configured")
	}
	if report.Telegram.Configured {
		t.Fatal("expected telegram not configured")
	}
}

func TestValidateChatWithoutTokenFails(t *testing.T) {
	cfg := &config.Config{
		Telegram: config.TelegramConfig{ChatID: "123"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when chat id set without token")
	}
}

func TestValidateTokenWithoutChatOK(t *testing.T) {
	cfg := &config.Config{
		Telegram: config.TelegramConfig{BotToken: "token"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.Telegram.Configured() {
		t.Fatal("expected telegram not fully configured without chat id")
	}
}

func TestValidatePartialSourceArenaEmptyOK(t *testing.T) {
	cfg := &config.Config{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
