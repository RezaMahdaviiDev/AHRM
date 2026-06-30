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

var tehranLocation = loadTehranLocation()

func loadTehranLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return time.FixedZone("IRST", 3*3600+30*60)
	}
	return loc
}

type pageData struct {
	Title             string
	Snapshot          scanner.Snapshot
	RefreshSeconds    int
	RefreshMinutes    int
	LastUpdatedTehran string
}

func (s *Server) registerPages(mux *http.ServeMux) {
	s.initTemplates()
	mux.HandleFunc("GET /dashboard", s.pageHandler("dashboard.html", "Dashboard"))
	mux.HandleFunc("GET /arbitrage", s.pageHandler("arbitrage.html", "Arbitrage"))
	mux.HandleFunc("GET /hv", s.pageHandler("hv.html", "HV"))
	mux.HandleFunc("GET /market", s.pageHandler("market.html", "Market"))
	mux.HandleFunc("GET /covered-call", s.pageHandler("covered-call.html", "Covered Call"))
	mux.HandleFunc("GET /matrix", s.pageHandler("matrix.html", "Matrix"))
	mux.HandleFunc("GET /bull-spread", s.pageHandler("bull-spread.html", "Bull Call Spread"))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
}

func (s *Server) initTemplates() {
	s.tplOnce.Do(func() {
		funcs := template.FuncMap{
			"mul100":   func(a int) int { return a * 100 },
			"divInt":   func(a, b int) int {
				if b == 0 { return 0 }
				return a / b
			},
			"toJalali": toJalali,
		}
		tpl, err := template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html")
		if err == nil {
			s.templates = tpl
		}
	})
}

func (s *Server) pageHandler(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := s.getSnapshot(r.Context())
		data := pageData{
			Title:             title,
			Snapshot:          snap,
			RefreshSeconds:    s.refreshSeconds,
			RefreshMinutes:    (s.refreshSeconds + 59) / 60,
			LastUpdatedTehran: formatTehran(snap.GeneratedAt),
		}
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

func formatTehran(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.In(tehranLocation).Format("2006/01/02 15:04:05")
}

func (s *Server) getSnapshot(ctx context.Context) scanner.Snapshot {
	if s.scanner == nil {
		return scanner.Snapshot{GeneratedAt: time.Now().UTC()}
	}
	ttl := s.cacheTTL()
	s.snapMu.RLock()
	if time.Since(s.snapAt) < ttl && !s.snapAt.IsZero() {
		snap := s.snapCache
		s.snapMu.RUnlock()
		return snap
	}
	s.snapMu.RUnlock()

	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	if time.Since(s.snapAt) < ttl && !s.snapAt.IsZero() {
		return s.snapCache
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	snap, _ := s.scanner.Refresh(ctx)
	s.snapCache = snap
	s.snapAt = time.Now()
	return snap
}
