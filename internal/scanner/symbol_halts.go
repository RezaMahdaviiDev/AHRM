package scanner

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"ahrm/internal/alerts"
	"ahrm/internal/boursecrawl"
	"ahrm/internal/market"
	"ahrm/internal/sourcearena"
)

const (
	symbolHaltDailySyncHour      = 8
	symbolHaltDailySyncMinute    = 0
	symbolHaltRefreshInterval    = 5 * time.Minute
	symbolHaltFetchMaxAttempts   = 3
	symbolHaltFetchRetryDelay    = 2 * time.Second
	reopenCheckInterval          = 2 * time.Minute
	tehranMarketOpenHour         = 9
	tehranMarketOpenMinute       = 0
	tehranMarketCloseHour        = 12
	tehranMarketCloseMinute      = 30
	defaultHaltReasonCategory    = "سایر"
	haltReasonCategoryAssembly   = "برگزاری مجمع"
	haltReasonCategoryDisclosure = "افشای اطلاعات با اهمیت"
	haltReasonCategoryVolatility = "نوسان قیمت / حراج"
	haltReasonCategoryClarify    = "شفاف‌سازی"
)

func (s *Service) StartSymbolHaltScheduler(ctx context.Context) {
	if s.client == nil || s.haltStore == nil {
		return
	}

	go func() {
		s.syncSymbolHaltsOnce(ctx)
		dailyTimer := time.NewTimer(time.Until(nextTehranTime(symbolHaltDailySyncHour, symbolHaltDailySyncMinute)))
		defer dailyTimer.Stop()
		refreshTicker := time.NewTicker(symbolHaltRefreshInterval)
		defer refreshTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-dailyTimer.C:
				s.syncSymbolHaltsOnce(ctx)
				dailyTimer.Reset(time.Until(nextTehranTime(symbolHaltDailySyncHour, symbolHaltDailySyncMinute)))
			case <-refreshTicker.C:
				// The daily sync runs before market open and can miss halts that occur
				// during the trading session, so keep refreshing while the market is open.
				if isTehranMarketHours() {
					s.syncSymbolHaltsOnce(ctx)
				}
			}
		}
	}()

	go func() {
		if isTehranMarketHours() {
			s.checkReopenAnnouncements(ctx)
		}
		ticker := time.NewTicker(reopenCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if isTehranMarketHours() {
					s.checkReopenAnnouncements(ctx)
				}
			}
		}
	}()
}

func (s *Service) syncSymbolHaltsOnce(ctx context.Context) {
	syncCtx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()

	closedSymbols, err := s.fetchClosedSymbolsWithRetry(syncCtx)
	if err != nil {
		slog.Error("symbol halt sync aborted: fetch closed symbols failed", "error", err)
		return
	}
	supervisorMessages, err := s.client.FetchSupervisorMessages(syncCtx)
	if err != nil {
		slog.Warn("symbol halt sync: fetch supervisor messages failed, continuing without them", "error", err)
		supervisorMessages = nil
	}
	latestHalts := latestMessagesBySymbol(supervisorMessages, isHaltSupervisorMessage)

	halts := make([]market.SymbolHalt, 0, len(closedSymbols))
	events := make([]market.SymbolHaltEvent, 0, len(closedSymbols)+len(supervisorMessages))
	for _, item := range closedSymbols {
		name := normalizeSymbol(item.Name)
		if name == "" {
			continue
		}
		latest := latestHalts[name]
		reason := strings.TrimSpace(item.Message)
		if reason == "" {
			reason = strings.TrimSpace(latest.Message)
		}
		fallbackSource := "sourcearena"
		if reason == "" {
			if fallback, ok := s.fetchHaltReasonFromBourse(syncCtx, name); ok {
				reason = fallback.Reason
				if haltedAt := strings.TrimSpace(fallback.PublishedAt); haltedAt != "" {
					item.HaltedAt = haltedAt
				}
				fallbackSource = "bourse-crawl"
			}
		}
		haltedAt := strings.TrimSpace(item.HaltedAt)
		if haltedAt == "" {
			haltedAt = strings.TrimSpace(latest.PublishedAt)
		}
		status := strings.TrimSpace(item.Status)
		if status == "" {
			status = "halted"
		}
		halts = append(halts, market.SymbolHalt{
			Name:                name,
			Status:              status,
			HaltCategory:        classifyHaltReason(reason),
			HaltReason:          reason,
			HaltedAt:            haltedAt,
			SupervisorMessage:   strings.TrimSpace(latest.Message),
			SupervisorMessageAt: strings.TrimSpace(latest.PublishedAt),
		})
		events = append(events, market.SymbolHaltEvent{
			Symbol:     name,
			EventType:  "halt",
			Reason:     reason,
			OccurredAt: haltedAt,
			Source:     fallbackSource,
			RawMessage: chooseFirstNonEmpty(item.Message, latest.Message),
		})
	}
	events = append(events, supervisorEvents(supervisorMessages)...)
	sort.Slice(halts, func(i, j int) bool { return halts[i].Name < halts[j].Name })
	if err := s.haltStore.ReplaceSymbolHalts(syncCtx, time.Now().UTC(), halts); err != nil {
		slog.Error("symbol halt sync: replace halts failed", "error", err)
		return
	}
	if err := s.haltStore.AppendSymbolHaltEvents(syncCtx, events); err != nil {
		slog.Error("symbol halt sync: append halt events failed", "error", err)
		return
	}
	slog.Info("symbol halt sync complete", "halted_count", len(halts), "event_count", len(events))
}

// fetchClosedSymbolsWithRetry retries transient failures from the closed_symbols
// endpoint, which has been observed to intermittently return an error payload.
func (s *Service) fetchClosedSymbolsWithRetry(ctx context.Context) ([]sourcearena.ClosedSymbol, error) {
	var lastErr error
	for attempt := 1; attempt <= symbolHaltFetchMaxAttempts; attempt++ {
		closed, err := s.client.FetchClosedSymbols(ctx)
		if err == nil {
			return closed, nil
		}
		lastErr = err
		slog.Warn("fetch closed symbols failed", "attempt", attempt, "max_attempts", symbolHaltFetchMaxAttempts, "error", err)
		if attempt < symbolHaltFetchMaxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(symbolHaltFetchRetryDelay):
			}
		}
	}
	return nil, lastErr
}

func (s *Service) checkReopenAnnouncements(ctx context.Context) {
	if s.alerts == nil {
		return
	}
	checkCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	_, halts, err := s.haltStore.LatestSymbolHalts(checkCtx)
	if err != nil {
		slog.Error("reopen check: read latest halts failed", "error", err)
		return
	}
	if len(halts) == 0 {
		return
	}
	haltedSet := make(map[string]struct{}, len(halts))
	for _, halt := range halts {
		haltedSet[normalizeSymbol(halt.Name)] = struct{}{}
	}

	messages, err := s.client.FetchSupervisorMessages(checkCtx)
	if err != nil {
		slog.Error("reopen check: fetch supervisor messages failed", "error", err)
		return
	}
	reopenMessages := latestMessagesBySymbol(messages, isReopenSupervisorMessage)
	events := make([]market.SymbolHaltEvent, 0, len(reopenMessages))
	for symbol, msg := range reopenMessages {
		if _, exists := haltedSet[symbol]; !exists {
			continue
		}
		_, _ = s.alerts.MaybeSendSymbolReopen(checkCtx, alerts.SymbolReopenAlertInput{
			Symbol:      symbol,
			PublishedAt: msg.PublishedAt,
			Message:     msg.Message,
		})
		events = append(events, market.SymbolHaltEvent{
			Symbol:     symbol,
			EventType:  "reopen",
			Reason:     msg.Message,
			OccurredAt: msg.PublishedAt,
			Source:     "sourcearena",
			RawMessage: msg.Message,
		})
	}
	if len(events) > 0 {
		if err := s.haltStore.AppendSymbolHaltEvents(checkCtx, events); err != nil {
			slog.Error("reopen check: append halt events failed", "error", err)
		}
	}
}

func latestMessagesBySymbol(messages []sourcearena.SupervisorMessage, keep func(string) bool) map[string]sourcearena.SupervisorMessage {
	out := make(map[string]sourcearena.SupervisorMessage)
	for _, item := range messages {
		symbol := normalizeSymbol(item.Symbol)
		if symbol == "" || !keep(item.Message) {
			continue
		}
		if _, exists := out[symbol]; exists {
			continue
		}
		out[symbol] = item
	}
	return out
}

func normalizeSymbol(symbol string) string {
	return strings.TrimSpace(strings.ReplaceAll(symbol, "\u200c", ""))
}

func classifyHaltReason(reason string) string {
	switch {
	case containsAny(reason, "مجمع", "افزایش سرمایه", "سالانه", "فوق العاده"):
		return haltReasonCategoryAssembly
	case containsAny(reason, "افشای اطلاعات", "گروه الف", "گروه ب"):
		return haltReasonCategoryDisclosure
	case containsAny(reason, "نوسان", "حراج", "دامنه", "مرحله پیش گشایش"):
		return haltReasonCategoryVolatility
	case containsAny(reason, "شفاف", "ابهام", "رفع ابهام", "توضیحات ناشر"):
		return haltReasonCategoryClarify
	default:
		return defaultHaltReasonCategory
	}
}

func isHaltSupervisorMessage(message string) bool {
	return containsAny(message, "متوقف", "توقف", "تعلیق", "عدم انجام معامله")
}

func isReopenSupervisorMessage(message string) bool {
	return containsAny(message,
		"آغاز بازگشایی",
		"آغاز دوره سفارش",
		"شروع بازگشایی",
		"بازگشایی نماد",
		"مرحله پیش گشایش",
		"آغاز حراج",
	)
}

func containsAny(message string, keywords ...string) bool {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return false
	}
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func isTehranMarketHours() bool {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		loc = time.FixedZone("IRST", 3*3600+30*60)
	}
	now := time.Now().In(loc)
	hour, minute := now.Hour(), now.Minute()
	afterOpen := hour > tehranMarketOpenHour || (hour == tehranMarketOpenHour && minute >= tehranMarketOpenMinute)
	beforeClose := hour < tehranMarketCloseHour || (hour == tehranMarketCloseHour && minute <= tehranMarketCloseMinute)
	return afterOpen && beforeClose
}

func (s *Service) fetchHaltReasonFromBourse(ctx context.Context, symbol string) (boursecrawl.Notice, bool) {
	if s.bourseCrawler == nil || !s.bourseCrawler.Enabled() {
		return boursecrawl.Notice{}, false
	}
	notice, err := s.bourseCrawler.FetchLatestNotice(ctx, symbol)
	if err != nil {
		return boursecrawl.Notice{}, false
	}
	if strings.TrimSpace(notice.Reason) == "" {
		return boursecrawl.Notice{}, false
	}
	return notice, true
}

func supervisorEvents(messages []sourcearena.SupervisorMessage) []market.SymbolHaltEvent {
	events := make([]market.SymbolHaltEvent, 0, len(messages))
	for _, msg := range messages {
		symbol := normalizeSymbol(msg.Symbol)
		if symbol == "" {
			continue
		}
		eventType := "supervisor"
		if isHaltSupervisorMessage(msg.Message) {
			eventType = "halt"
		} else if isReopenSupervisorMessage(msg.Message) {
			eventType = "reopen"
		}
		events = append(events, market.SymbolHaltEvent{
			Symbol:     symbol,
			EventType:  eventType,
			Reason:     msg.Message,
			OccurredAt: msg.PublishedAt,
			Source:     "sourcearena",
			RawMessage: msg.Message,
		})
	}
	return events
}

func chooseFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
