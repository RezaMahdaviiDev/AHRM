package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr    string
	LogLevel    string
	Supabase    SupabaseConfig
	SourceArena SourceArenaConfig
	Telegram    TelegramConfig
	Alerts      AlertsConfig
}

type AlertsConfig struct {
	ArbitrageRThreshold  float64
	BreadthHighThreshold float64
	BreadthLowThreshold  float64
	AdvanceHighThreshold float64
	AdvanceLowThreshold  float64
}

type SupabaseConfig struct {
	Enabled  bool
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type SourceArenaConfig struct {
	APIToken  string
	HTTPProxy string
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type ServiceStatus struct {
	Configured bool `json:"configured"`
	Connected  bool `json:"connected,omitempty"`
}

type Readiness struct {
	ConfigLoaded bool          `json:"config_loaded"`
	Supabase     ServiceStatus `json:"supabase"`
	SourceArena  ServiceStatus `json:"sourcearena"`
	Telegram     ServiceStatus `json:"telegram"`
}

func (s SupabaseConfig) Configured() bool {
	if !s.Enabled {
		return false
	}
	return anyNonEmpty(s.Host, s.Port, s.Name, s.User, s.Password, s.SSLMode)
}

func (s SourceArenaConfig) Configured() bool {
	return strings.TrimSpace(s.APIToken) != ""
}

func (t TelegramConfig) Configured() bool {
	return strings.TrimSpace(t.BotToken) != "" && strings.TrimSpace(t.ChatID) != ""
}

func Load() (*Config, error) {
	loadEnvFiles()
	return LoadFromEnv()
}

func loadEnvFiles() {
	_ = godotenv.Load()
	if os.Getenv("SUPABASE_DB_HOST") != "" {
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
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),
		Supabase: SupabaseConfig{
			Enabled:  parseBoolEnv("SUPABASE_ENABLED", false),
			Host:     strings.TrimSpace(os.Getenv("SUPABASE_DB_HOST")),
			Port:     strings.TrimSpace(os.Getenv("SUPABASE_DB_PORT")),
			Name:     strings.TrimSpace(os.Getenv("SUPABASE_DB_NAME")),
			User:     strings.TrimSpace(os.Getenv("SUPABASE_DB_USER")),
			Password: os.Getenv("SUPABASE_DB_PASSWORD"),
			SSLMode:  strings.TrimSpace(os.Getenv("SUPABASE_DB_SSLMODE")),
		},
		SourceArena: SourceArenaConfig{
			APIToken:  strings.TrimSpace(os.Getenv("SOURCEARENA_API_TOKEN")),
			HTTPProxy: strings.TrimSpace(os.Getenv("SOURCEARENA_HTTP_PROXY")),
		},
		Telegram: TelegramConfig{
			BotToken: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
			ChatID:   strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")),
		},
		Alerts: AlertsConfig{
			ArbitrageRThreshold:  parseFloatEnv("ALERT_ARBITRAGE_R_THRESHOLD", 0),
			BreadthHighThreshold: parseFloatEnv("ALERT_BREADTH_HIGH", 0.618),
			BreadthLowThreshold:  parseFloatEnv("ALERT_BREADTH_LOW", 0.4),
			AdvanceHighThreshold: parseFloatEnv("ALERT_ADVANCE_HIGH", 2.0),
			AdvanceLowThreshold:  parseFloatEnv("ALERT_ADVANCE_LOW", 0.8),
		},
	}
	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Supabase.Enabled {
		if err := validateGroup("supabase", c.Supabase.Configured(), map[string]string{
			"SUPABASE_DB_HOST":     c.Supabase.Host,
			"SUPABASE_DB_PORT":     c.Supabase.Port,
			"SUPABASE_DB_NAME":     c.Supabase.Name,
			"SUPABASE_DB_USER":     c.Supabase.User,
			"SUPABASE_DB_PASSWORD": c.Supabase.Password,
			"SUPABASE_DB_SSLMODE":  c.Supabase.SSLMode,
		}); err != nil {
			return err
		}
	}
	if err := validateGroup("sourcearena", c.SourceArena.Configured(), map[string]string{
		"SOURCEARENA_API_TOKEN": c.SourceArena.APIToken,
	}); err != nil {
		return err
	}
	if err := c.validateTelegram(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateTelegram() error {
	hasToken := strings.TrimSpace(c.Telegram.BotToken) != ""
	hasChat := strings.TrimSpace(c.Telegram.ChatID) != ""
	if hasChat && !hasToken {
		return fmt.Errorf("telegram: TELEGRAM_BOT_TOKEN is required when TELEGRAM_CHAT_ID is set")
	}
	return nil
}

func (c *Config) ReadinessReport(dbConnected bool) Readiness {
	return Readiness{
		ConfigLoaded: true,
		Supabase: ServiceStatus{
			Configured: c.Supabase.Configured(),
			Connected:  c.Supabase.Configured() && dbConnected,
		},
		SourceArena: ServiceStatus{Configured: c.SourceArena.Configured()},
		Telegram:    ServiceStatus{Configured: c.Telegram.Configured()},
	}
}

func (c *Config) SupabaseDSN() (string, error) {
	if !c.Supabase.Configured() {
		return "", fmt.Errorf("supabase is not configured")
	}
	port := c.Supabase.Port
	if port == "" {
		port = "5432"
	}
	sslMode := c.Supabase.SSLMode
	if sslMode == "" {
		sslMode = "require"
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Supabase.User, c.Supabase.Password),
		Host:   fmt.Sprintf("%s:%s", c.Supabase.Host, port),
		Path:   c.Supabase.Name,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()
	return u.String(), nil
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

func anyNonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
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
