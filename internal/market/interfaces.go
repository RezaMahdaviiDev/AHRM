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
