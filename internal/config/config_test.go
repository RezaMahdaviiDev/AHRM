package config_test

import (
	"testing"

	"ahrm/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SOURCEARENA_API_TOKEN", "")
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

func TestReadinessReportUnconfigured(t *testing.T) {
	cfg := &config.Config{}
	report := cfg.ReadinessReport()
	if !report.ConfigLoaded {
		t.Fatal("expected config_loaded true")
	}
	if report.SourceArena.Configured {
		t.Fatal("expected sourcearena not configured")
	}
}

func TestValidatePartialSourceArenaEmptyOK(t *testing.T) {
	cfg := &config.Config{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateBaleRequiresTokenWithChatIDs(t *testing.T) {
	cfg := &config.Config{
		Bale: config.BaleConfig{
			BotToken: "",
			ChatIDs:  "12345",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when ChatIDs set but BotToken missing")
	}
}

func TestValidateBaleOKWhenBothSet(t *testing.T) {
	cfg := &config.Config{
		Bale: config.BaleConfig{
			BotToken: "bot:token",
			ChatIDs:  "12345",
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateBourseCrawlTemplateRequiresSymbolPlaceholder(t *testing.T) {
	cfg := &config.Config{
		BourseCrawlURLTemplate: "https://example.com/notice",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing {symbol}")
	}
}
