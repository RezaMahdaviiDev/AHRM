package market

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"ahrm/internal/indicators"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS market_daily_stats (
		date     TEXT PRIMARY KEY,
		positive INTEGER NOT NULL DEFAULT 0,
		negative INTEGER NOT NULL DEFAULT 0,
		total    INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) UpsertToday(ctx context.Context, day indicators.DailyMarket) error {
	today := time.Now().UTC().Format("2006-01-02")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO market_daily_stats (date, positive, negative, total)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			positive = excluded.positive,
			negative = excluded.negative,
			total    = excluded.total`,
		today, day.Positive, day.Negative, day.Total,
	)
	return err
}

func (s *SQLiteStore) UpsertDay(ctx context.Context, date time.Time, day indicators.DailyMarket) error {
	dateStr := date.UTC().Format("2006-01-02")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO market_daily_stats (date, positive, negative, total)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			positive = excluded.positive,
			negative = excluded.negative,
			total    = excluded.total`,
		dateStr, day.Positive, day.Negative, day.Total,
	)
	return err
}

func (s *SQLiteStore) LastDays(ctx context.Context, days int) ([]indicators.DailyMarket, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date, positive, negative, total
		FROM market_daily_stats
		ORDER BY date DESC
		LIMIT ?`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []indicators.DailyMarket
	for rows.Next() {
		var d indicators.DailyMarket
		if err := rows.Scan(&d.Date, &d.Positive, &d.Negative, &d.Total); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// reverse to chronological order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *SQLiteStore) ExistingDays(ctx context.Context, from, to time.Time) (map[string]struct{}, error) {
	fromStr := from.UTC().Format("2006-01-02")
	toStr := to.UTC().Format("2006-01-02")
	rows, err := s.db.QueryContext(ctx, `
		SELECT date FROM market_daily_stats
		WHERE date >= ? AND date <= ?`, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out[d] = struct{}{}
	}
	return out, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
