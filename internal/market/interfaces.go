package market

import (
	"context"
	"time"

	"ahrm/internal/indicators"
)

type DailyStore interface {
	UpsertToday(ctx context.Context, day indicators.DailyMarket) error
	UpsertDay(ctx context.Context, date time.Time, day indicators.DailyMarket) error
	LastDays(ctx context.Context, days int) ([]indicators.DailyMarket, error)
	ExistingDays(ctx context.Context, from, to time.Time) (map[string]struct{}, error)
}

type SymbolSnapshotStore interface {
	UpsertSymbolSnapshot(ctx context.Context, date string, rows []indicators.SymbolRow) error
	LatestSymbolSnapshot(ctx context.Context) (date string, rows []indicators.SymbolRow, err error)
}

type SymbolRegistryStore interface {
	RegisterSymbols(ctx context.Context, names []string) ([]string, error)
	UpsertQueueStreaks(ctx context.Context, names []string) (map[string]int, error)
}

type SymbolHalt struct {
	Name                string `json:"name"`
	Status              string `json:"status"`
	HaltCategory        string `json:"halt_category"`
	HaltReason          string `json:"halt_reason"`
	HaltedAt            string `json:"halted_at"`
	SupervisorMessage   string `json:"supervisor_message"`
	SupervisorMessageAt string `json:"supervisor_message_at"`
}

type SymbolHaltEvent struct {
	Symbol     string `json:"symbol"`
	EventType  string `json:"event_type"`
	Reason     string `json:"reason"`
	OccurredAt string `json:"occurred_at"`
	Source     string `json:"source"`
	RawMessage string `json:"raw_message"`
}

type SymbolHaltStore interface {
	ReplaceSymbolHalts(ctx context.Context, checkedAt time.Time, halts []SymbolHalt) error
	LatestSymbolHalts(ctx context.Context) (checkedAt time.Time, halts []SymbolHalt, err error)
	AppendSymbolHaltEvents(ctx context.Context, events []SymbolHaltEvent) error
	RecentSymbolHaltEvents(ctx context.Context, limit int) ([]SymbolHaltEvent, error)
}
