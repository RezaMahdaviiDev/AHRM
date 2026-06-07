package server

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/scanner"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg        *config.Config
	pool       *pgxpool.Pool
	logger     *slog.Logger
	migrations string
	dbReady    bool
	scanner    *scanner.Service
	templates  *template.Template
	tplOnce    sync.Once
	snapMu     sync.RWMutex
	snapCache  scanner.Snapshot
	snapAt     time.Time
}

func New(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger, migrationsDir string, dbReady bool, scan *scanner.Service) *Server {
	return &Server{
		cfg:        cfg,
		pool:       pool,
		logger:     logger,
		migrations: migrationsDir,
		dbReady:    dbReady,
		scanner:    scan,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)
	if s.scanner != nil {
		s.registerPages(mux)
	}
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
