package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr    string
	LogLevel    string
	Supabase    SupabaseConfig
	SourceArena SourceArenaConfig
	Telegram    TelegramConfig
}

type SupabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type SourceArenaConfig struct {
	APIToken string
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
	return anyNonEmpty(s.Host, s.Port, s.Name, s.User, s.Password, s.SSLMode)
}

func (s SourceArenaConfig) Configured() bool {
	return strings.TrimSpace(s.APIToken) != ""
}

func (t TelegramConfig) Configured() bool {
	return anyNonEmpty(t.BotToken, t.ChatID)
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	return LoadFromEnv()
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		HTTPAddr: getenv("HTTP_ADDR", ":8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),
		Supabase: SupabaseConfig{
			Host:     strings.TrimSpace(os.Getenv("SUPABASE_DB_HOST")),
			Port:     strings.TrimSpace(os.Getenv("SUPABASE_DB_PORT")),
			Name:     strings.TrimSpace(os.Getenv("SUPABASE_DB_NAME")),
			User:     strings.TrimSpace(os.Getenv("SUPABASE_DB_USER")),
			Password: os.Getenv("SUPABASE_DB_PASSWORD"),
			SSLMode:  strings.TrimSpace(os.Getenv("SUPABASE_DB_SSLMODE")),
		},
		SourceArena: SourceArenaConfig{
			APIToken: strings.TrimSpace(os.Getenv("SOURCEARENA_API_TOKEN")),
		},
		Telegram: TelegramConfig{
			BotToken: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
			ChatID:   strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")),
		},
	}
	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
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
	if err := validateGroup("sourcearena", c.SourceArena.Configured(), map[string]string{
		"SOURCEARENA_API_TOKEN": c.SourceArena.APIToken,
	}); err != nil {
		return err
	}
	if err := validateGroup("telegram", c.Telegram.Configured(), map[string]string{
		"TELEGRAM_BOT_TOKEN": c.Telegram.BotToken,
		"TELEGRAM_CHAT_ID":   c.Telegram.ChatID,
	}); err != nil {
		return err
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
