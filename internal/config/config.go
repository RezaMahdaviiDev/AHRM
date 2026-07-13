package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr               string
	LogLevel               string
	MatrixAlertsFile       string
	BourseCrawlURLTemplate string
	BourseCrawlUserAgent   string
	RiskFreeRate           float64
	SnapshotRefreshSeconds int
	SourceArena            SourceArenaConfig
	Bale                   BaleConfig
	Alerts                 AlertsConfig
}

type AlertsConfig struct {
	ArbitrageRThreshold     float64
	ArbitrageR12Threshold   float64
	BreadthHighThreshold    float64
	BreadthLowThreshold     float64
	AdvanceHighThreshold    float64
	AdvanceLowThreshold     float64
	CoveredCallROIThreshold float64
	BullSpreadATMThreshold     float64
	BullSpreadOTMThreshold     float64
	BearPutSpreadATMThreshold  float64
	BearPutSpreadOTMThreshold  float64
}

type SourceArenaConfig struct {
	APIToken  string
	HTTPProxy string
}

type BaleConfig struct {
	BotToken string
	ChatIDs  string // comma-separated list of chat IDs
}

type ServiceStatus struct {
	Configured bool `json:"configured"`
}

type Readiness struct {
	ConfigLoaded bool          `json:"config_loaded"`
	SourceArena  ServiceStatus `json:"sourcearena"`
}

func (s SourceArenaConfig) Configured() bool {
	return strings.TrimSpace(s.APIToken) != ""
}

func (b BaleConfig) Configured() bool {
	return strings.TrimSpace(b.BotToken) != "" && strings.TrimSpace(b.ChatIDs) != ""
}

func Load() (*Config, error) {
	loadEnvFiles()
	return LoadFromEnv()
}

func loadEnvFiles() {
	_ = godotenv.Load()
	if os.Getenv("HTTP_ADDR") != "" {
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		envPath := filepath.Join(dir, ".env")
		if _, statErr := os.Stat(envPath); statErr == nil {
			_ = godotenv.Load(envPath)
			return
		}
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		LogLevel:               getenv("LOG_LEVEL", "info"),
		MatrixAlertsFile:       getenv("MATRIX_ALERTS_FILE", "configs/matrix_alerts.json"),
		BourseCrawlURLTemplate: strings.TrimSpace(os.Getenv("BOURSE_CRAWL_URL_TEMPLATE")),
		BourseCrawlUserAgent:   getenv("BOURSE_CRAWL_USER_AGENT", "AHRM/1.0 (+symbol-halt-fallback-crawler)"),
		RiskFreeRate:           parseFloatEnv("RISK_FREE_RATE", 0.20),
		SnapshotRefreshSeconds: parseIntEnv("SNAPSHOT_REFRESH_SECONDS", 180),
		SourceArena: SourceArenaConfig{
			APIToken:  strings.TrimSpace(os.Getenv("SOURCEARENA_API_TOKEN")),
			HTTPProxy: strings.TrimSpace(os.Getenv("SOURCEARENA_HTTP_PROXY")),
		},
		Bale: BaleConfig{
			BotToken: strings.TrimSpace(os.Getenv("BALE_BOT_TOKEN")),
			ChatIDs:  strings.TrimSpace(os.Getenv("BALE_CHAT_IDS")),
		},
		Alerts: AlertsConfig{
			ArbitrageRThreshold:     parseFloatEnv("ALERT_ARBITRAGE_R_THRESHOLD", 0),
			ArbitrageR12Threshold:   parseFloatEnv("ALERT_ARBITRAGE_R12_THRESHOLD", 10.0),
			BreadthHighThreshold:    parseFloatEnv("ALERT_BREADTH_HIGH", 0.618),
			BreadthLowThreshold:     parseFloatEnv("ALERT_BREADTH_LOW", 0.4),
			AdvanceHighThreshold:    parseFloatEnv("ALERT_ADVANCE_HIGH", 1.4),
			AdvanceLowThreshold:     parseFloatEnv("ALERT_ADVANCE_LOW", 0.6),
			CoveredCallROIThreshold: parseFloatEnv("ALERT_COVERED_CALL_ROI_THRESHOLD", 30.0),
			BullSpreadATMThreshold:     parseFloatEnv("ALERT_BULL_SPREAD_ATM_THRESHOLD", 2.0),
			BullSpreadOTMThreshold:     parseFloatEnv("ALERT_BULL_SPREAD_OTM_THRESHOLD", 3.0),
			BearPutSpreadATMThreshold:  parseFloatEnv("ALERT_BEAR_PUT_SPREAD_ATM_THRESHOLD", 1.5),
			BearPutSpreadOTMThreshold:  parseFloatEnv("ALERT_BEAR_PUT_SPREAD_OTM_THRESHOLD", 3.0),
		},
	}
	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.BourseCrawlURLTemplate != "" && !strings.Contains(c.BourseCrawlURLTemplate, "{symbol}") {
		return fmt.Errorf("bourse crawl: BOURSE_CRAWL_URL_TEMPLATE must include {symbol} placeholder")
	}
	if err := validateGroup("sourcearena", c.SourceArena.Configured(), map[string]string{
		"SOURCEARENA_API_TOKEN": c.SourceArena.APIToken,
	}); err != nil {
		return err
	}
	return c.validateBale()
}

func (c *Config) validateBale() error {
	hasToken := strings.TrimSpace(c.Bale.BotToken) != ""
	hasChat := strings.TrimSpace(c.Bale.ChatIDs) != ""
	if hasChat && !hasToken {
		return fmt.Errorf("bale: BALE_BOT_TOKEN is required when BALE_CHAT_IDS is set")
	}
	return nil
}

func (c *Config) ReadinessReport() Readiness {
	return Readiness{
		ConfigLoaded: true,
		SourceArena:  ServiceStatus{Configured: c.SourceArena.Configured()},
	}
}

func validateGroup(name string, configured bool, fields map[string]string) error {
	if !configured {
		for _, value := range fields {
			if strings.TrimSpace(value) != "" {
				return fmt.Errorf("%s: incomplete configuration; provide all related environment variables or leave them empty", name)
			}
		}
		return nil
	}
	for envName, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: %s is required when %s is configured", name, envName, name)
		}
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseFloatEnv(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var value float64
	if _, err := fmt.Sscanf(raw, "%f", &value); err != nil {
		return fallback
	}
	return value
}

func parseIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return fallback
	}
	return value
}
