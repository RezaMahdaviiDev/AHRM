package server

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/scanner"
)

type Server struct {
	cfg             *config.Config
	logger          *slog.Logger
	scanner         *scanner.Service
	templates       *template.Template
	tplOnce         sync.Once
	snapMu          sync.RWMutex
	snapCache       scanner.Snapshot
	snapAt          time.Time
	refreshInterval time.Duration
	refreshSeconds  int
}

func New(cfg *config.Config, logger *slog.Logger, scan *scanner.Service) *Server {
	secs := cfg.SnapshotRefreshSeconds
	if secs <= 0 {
		secs = 180
	}
	return &Server{
		cfg:             cfg,
		logger:          logger,
		scanner:         scan,
		refreshInterval: time.Duration(secs) * time.Second,
		refreshSeconds:  secs,
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
	report := s.cfg.ReadinessReport()
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) ReadinessReport() config.Readiness {
	return s.cfg.ReadinessReport()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
