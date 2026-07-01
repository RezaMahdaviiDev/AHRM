package scanner

import (
	"context"
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
		for {
			next := nextTehranTime(symbolHaltDailySyncHour, symbolHaltDailySyncMinute)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(next)):
				s.syncSymbolHaltsOnce(ctx)
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
	syncCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	closedSymbols, err := s.client.FetchClosedSymbols(syncCtx)
	if err != nil {
		return
	}
	supervisorMessages, err := s.client.FetchSupervisorMessages(syncCtx)
	if err != nil {
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
	_ = s.haltStore.ReplaceSymbolHalts(syncCtx, time.Now().UTC(), halts)
	_ = s.haltStore.AppendSymbolHaltEvents(syncCtx, events)
}

func (s *Service) checkReopenAnnouncements(ctx context.Context) {
	if s.alerts == nil {
		return
	}
	checkCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	_, halts, err := s.haltStore.LatestSymbolHalts(checkCtx)
	if err != nil || len(halts) == 0 {
		return
	}
	haltedSet := make(map[string]struct{}, len(halts))
	for _, halt := range halts {
		haltedSet[normalizeSymbol(halt.Name)] = struct{}{}
	}

	messages, err := s.client.FetchSupervisorMessages(checkCtx)
	if err != nil {
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
	_ = s.haltStore.AppendSymbolHaltEvents(checkCtx, events)
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
