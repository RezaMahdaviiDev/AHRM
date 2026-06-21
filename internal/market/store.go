package market

import (
	"context"
	"time"

	"ahrm/internal/indicators"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DailyStore interface {
	UpsertToday(ctx context.Context, day indicators.DailyMarket) error
	UpsertDay(ctx context.Context, date time.Time, day indicators.DailyMarket) error
	LastDays(ctx context.Context, days int) ([]indicators.DailyMarket, error)
	// ExistingDays returns the set of dates (formatted as YYYY-MM-DD) already
	// stored within the inclusive [from, to] range. Used to decide whether the
	// expensive per-symbol candle backfill can be skipped.
	ExistingDays(ctx context.Context, from, to time.Time) (map[string]struct{}, error)
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) UpsertToday(ctx context.Context, day indicators.DailyMarket) error {
	if s == nil || s.pool == nil {
		return nil
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO market_daily_stats (day, positive, negative, total)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (day) DO UPDATE SET positive=EXCLUDED.positive, negative=EXCLUDED.negative, total=EXCLUDED.total`,
		today, day.Positive, day.Negative, day.Total,
	)
	return err
}

func (s *Store) UpsertDay(ctx context.Context, date time.Time, day indicators.DailyMarket) error {
	if s == nil || s.pool == nil {
		return nil
	}
	d := date.UTC().Truncate(24 * time.Hour)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO market_daily_stats (day, positive, negative, total)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (day) DO UPDATE SET positive=EXCLUDED.positive, negative=EXCLUDED.negative, total=EXCLUDED.total`,
		d, day.Positive, day.Negative, day.Total,
	)
	return err
}

func (s *Store) LastDays(ctx context.Context, days int) ([]indicators.DailyMarket, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT positive, negative, total
		FROM market_daily_stats
		ORDER BY day DESC
		LIMIT $1`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []indicators.DailyMarket
	for rows.Next() {
		var day indicators.DailyMarket
		if err := rows.Scan(&day.Positive, &day.Negative, &day.Total); err != nil {
			return nil, err
		}
		out = append(out, day)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *Store) ExistingDays(ctx context.Context, from, to time.Time) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if s == nil || s.pool == nil {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT day
		FROM market_daily_stats
		WHERE day >= $1 AND day <= $2`,
		from.UTC().Truncate(24*time.Hour), to.UTC().Truncate(24*time.Hour),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var day time.Time
		if err := rows.Scan(&day); err != nil {
			return nil, err
		}
		out[day.UTC().Format("2006-01-02")] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
