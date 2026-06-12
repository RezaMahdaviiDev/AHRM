package server

import (
	"context"
	"time"

	"ahrm/internal/scanner"
)

func (s *Server) StartBackgroundRefresh(ctx context.Context) {
	if s.scanner == nil || s.refreshInterval <= 0 {
		return
	}
	go func() {
		s.runRefresh(context.Background())
		ticker := time.NewTicker(s.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runRefresh(context.Background())
			}
		}
	}()
}

func (s *Server) runRefresh(ctx context.Context) {
	if s.scanner == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	snap, _ := s.scanner.Refresh(ctx)
	s.setSnapshot(snap)
	if s.logger != nil && len(snap.Errors) > 0 {
		s.logger.Warn("snapshot refresh completed with errors", "count", len(snap.Errors))
	}
}

func (s *Server) setSnapshot(snap scanner.Snapshot) {
	s.snapMu.Lock()
	s.snapCache = snap
	s.snapAt = time.Now()
	s.snapMu.Unlock()
}

func (s *Server) cacheTTL() time.Duration {
	if s.refreshInterval > 0 {
		return s.refreshInterval
	}
	return 60 * time.Second
}
