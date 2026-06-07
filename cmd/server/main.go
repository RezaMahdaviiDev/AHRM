package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/db"
	"ahrm/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	level := parseLogLevel(cfg.LogLevel)
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	if pool != nil {
		defer pool.Close()
		if err := db.Ping(ctx, pool); err != nil {
			logger.Error("database ping failed", "error", err)
			os.Exit(1)
		}
		migrationsDir := filepath.Join(projectRoot(), "migrations")
		if err := db.Migrate(ctx, pool, migrationsDir); err != nil {
			logger.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	dbReady := pool != nil
	srv := server.New(cfg, pool, logger, filepath.Join(projectRoot(), "migrations"), dbReady)
	logReadiness(logger, srv.ReadinessReport())

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("server error", "error", serveErr)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func logReadiness(logger *slog.Logger, report config.Readiness) {
	payload, err := json.Marshal(report)
	if err != nil {
		logger.Info("startup readiness", "report", report)
		return
	}
	logger.Info("startup readiness", "report", json.RawMessage(payload))
}

func parseLogLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func projectRoot() string {
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return root
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
