package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ahrm/internal/alerts"
	"ahrm/internal/bale"
	"ahrm/internal/config"
	"ahrm/internal/market"
	"ahrm/internal/scanner"
	"ahrm/internal/server"
	"ahrm/internal/sourcearena"
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

	var saClient *sourcearena.Client
	if cfg.SourceArena.Configured() {
		saClient = sourcearena.NewClient(cfg.SourceArena, sourcearena.NopRawStore{})
	}

	var baleSender alerts.MessageSender
	if cfg.Bale.Configured() {
		baleSender = bale.NewClient(cfg.Bale)
	}

	alertStore, err := alerts.NewSQLiteStore(filepath.Join(projectRoot(), "data", "alerts.db"))
	if err != nil {
		logger.Error("alert store init failed", "error", err)
		os.Exit(1)
	}

	alertEngine := alerts.NewEngine(alerts.Config{
		ArbitrageRThreshold:     cfg.Alerts.ArbitrageRThreshold,
		ArbitrageR12Threshold:   cfg.Alerts.ArbitrageR12Threshold,
		BreadthHighThreshold:    cfg.Alerts.BreadthHighThreshold,
		BreadthLowThreshold:     cfg.Alerts.BreadthLowThreshold,
		AdvanceHighThreshold:    cfg.Alerts.AdvanceHighThreshold,
		AdvanceLowThreshold:     cfg.Alerts.AdvanceLowThreshold,
		CoveredCallROIThreshold: cfg.Alerts.CoveredCallROIThreshold,
		BullSpreadATMThreshold:  cfg.Alerts.BullSpreadATMThreshold,
		BullSpreadOTMThreshold:  cfg.Alerts.BullSpreadOTMThreshold,
	}, baleSender, alertStore)

	sqliteStore, err := market.NewSQLiteStore(filepath.Join(projectRoot(), "data", "market.db"))
	if err != nil {
		logger.Error("market store init failed", "error", err)
		os.Exit(1)
	}

	scan := scanner.NewService(cfg, saClient, sqliteStore, sqliteStore, sqliteStore, sqliteStore, alertEngine)

	srv := server.New(cfg, logger, scan)
	alertEngine.SetOnAlert(srv.Broadcaster().Publish)

	refreshCtx, stopRefresh := context.WithCancel(context.Background())
	defer stopRefresh()
	srv.StartBackgroundRefresh(refreshCtx)
	scan.StartBackfillScheduler(refreshCtx)
	scan.StartSymbolHaltScheduler(refreshCtx)

	logReadiness(logger, srv.ReadinessReport())

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("cannot bind HTTP port",
			"addr", cfg.HTTPAddr,
			"error", err,
			"hint", portInUseHint(cfg.HTTPAddr),
		)
		os.Exit(1)
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if serveErr := httpServer.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("server error", "error", serveErr)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	stopRefresh()

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

func portInUseHint(addr string) string {
	port := addr
	if strings.HasPrefix(addr, ":") {
		port = strings.TrimPrefix(addr, ":")
	}
	return fmt.Sprintf(
		"port %s may already be in use — PowerShell: Get-NetTCPConnection -LocalPort %s -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }",
		port, port,
	)
}
