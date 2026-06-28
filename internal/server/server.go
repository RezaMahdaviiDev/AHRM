package server

import (
	"encoding/json"
	_ "embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/scanner"
)

//go:embed static/alerts.js
var alertsJS []byte

type Server struct {
	cfg             *config.Config
	logger          *slog.Logger
	scanner         *scanner.Service
	broadcaster     *Broadcaster
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
		broadcaster:     NewBroadcaster(),
		refreshInterval: time.Duration(secs) * time.Second,
		refreshSeconds:  secs,
	}
}

// Broadcaster returns the server's event broadcaster so callers can wire alert callbacks.
func (s *Server) Broadcaster() *Broadcaster { return s.broadcaster }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("GET /static/alerts.js", s.handleAlertsJS)
	mux.HandleFunc("GET /test-alert", func(w http.ResponseWriter, _ *http.Request) {
		s.broadcaster.Publish("🔔 تست الارم — اگر این پیام را می‌بینید، سیستم نوتیف کار می‌کند")
		_, _ = w.Write([]byte("ok"))
	})
	if s.scanner != nil {
		s.registerPages(mux)
	}
	return mux
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush() // send headers immediately so the client unblocks
	ch := s.broadcaster.Subscribe()
	defer s.broadcaster.Unsubscribe(ch)
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleAlertsJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(alertsJS)
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
