package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"ahrm/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg        *config.Config
	pool       *pgxpool.Pool
	logger     *slog.Logger
	migrations string
	dbReady    bool
}

func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger, migrationsDir string, dbReady bool) *Server {
	return &Server{
		cfg:        cfg,
		pool:       pool,
		logger:     logger,
		migrations: migrationsDir,
		dbReady:    dbReady,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	report := s.cfg.ReadinessReport(s.dbReady)
	status := http.StatusOK

	if s.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.pool.Ping(ctx); err != nil {
			report.Supabase.Connected = false
			status = http.StatusServiceUnavailable
		} else {
			report.Supabase.Connected = true
		}
	}

	writeJSON(w, status, report)
}

func (s *Server) ReadinessReport() config.Readiness {
	return s.cfg.ReadinessReport(s.dbReady)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
