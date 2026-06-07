package server

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"time"

	"ahrm/internal/scanner"
)

//go:embed templates/*.html
var templateFS embed.FS

type pageData struct {
	Title    string
	Snapshot scanner.Snapshot
}

func (s *Server) registerPages(mux *http.ServeMux) {
	s.initTemplates()
	mux.HandleFunc("GET /dashboard", s.pageHandler("dashboard.html", "Dashboard"))
	mux.HandleFunc("GET /arbitrage", s.pageHandler("arbitrage.html", "Arbitrage"))
	mux.HandleFunc("GET /hv", s.pageHandler("hv.html", "HV"))
	mux.HandleFunc("GET /market", s.pageHandler("market.html", "Market"))
	mux.HandleFunc("GET /matrix", s.pageHandler("matrix.html", "Matrix"))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
}

func (s *Server) initTemplates() {
	s.tplOnce.Do(func() {
		tpl, err := template.ParseFS(templateFS, "templates/*.html")
		if err == nil {
			s.templates = tpl
		}
	})
}

func (s *Server) pageHandler(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := s.getSnapshot(r.Context())
		data := pageData{Title: title, Snapshot: snap}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if s.templates == nil {
			http.Error(w, "templates not loaded", http.StatusInternalServerError)
			return
		}
		if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) getSnapshot(ctx context.Context) scanner.Snapshot {
	if s.scanner == nil {
		return scanner.Snapshot{GeneratedAt: time.Now().UTC()}
	}
	s.snapMu.RLock()
	if time.Since(s.snapAt) < 60*time.Second && !s.snapAt.IsZero() {
		snap := s.snapCache
		s.snapMu.RUnlock()
		return snap
	}
	s.snapMu.RUnlock()

	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	if time.Since(s.snapAt) < 60*time.Second && !s.snapAt.IsZero() {
		return s.snapCache
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	snap, _ := s.scanner.Refresh(ctx)
	s.snapCache = snap
	s.snapAt = time.Now()
	return snap
}
