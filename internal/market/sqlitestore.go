package market

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS market_symbol_snapshot (
		snapshot_date TEXT NOT NULL,
		name          TEXT NOT NULL,
		change_pct    REAL NOT NULL,
		status        TEXT NOT NULL,
		PRIMARY KEY (snapshot_date, name)
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS symbol_registry (
		name            TEXT PRIMARY KEY,
		first_seen_date TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS queue_streak (
		name      TEXT PRIMARY KEY,
		streak    INTEGER NOT NULL DEFAULT 1,
		last_date TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	// Remove any zero-total rows written during market holidays (Scenario B).
	db.Exec(`DELETE FROM market_daily_stats WHERE total = 0`)
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

func (s *SQLiteStore) UpsertSymbolSnapshot(ctx context.Context, date string, rows []indicators.SymbolRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err = tx.ExecContext(ctx, `DELETE FROM market_symbol_snapshot WHERE snapshot_date = ?`, date); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO market_symbol_snapshot (snapshot_date, name, change_pct, status) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err = stmt.ExecContext(ctx, date, r.Name, r.ChangePct, r.Status); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) LatestSymbolSnapshot(ctx context.Context) (date string, rows []indicators.SymbolRow, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT snapshot_date FROM market_symbol_snapshot ORDER BY snapshot_date DESC LIMIT 1`).Scan(&date)
	if err != nil {
		return "", nil, err
	}
	qrows, err := s.db.QueryContext(ctx, `SELECT name, change_pct, status FROM market_symbol_snapshot WHERE snapshot_date = ? ORDER BY change_pct DESC`, date)
	if err != nil {
		return date, nil, err
	}
	defer qrows.Close()
	for qrows.Next() {
		var r indicators.SymbolRow
		if err = qrows.Scan(&r.Name, &r.ChangePct, &r.Status); err != nil {
			return date, nil, err
		}
		rows = append(rows, r)
	}
	return date, rows, qrows.Err()
}

// UpsertQueueStreaks updates consecutive-day streaks for symbols currently in buy queue.
// A streak continues when the symbol was last seen within 4 calendar days (covers Fri+Sat off).
// Returns a map of name → streak count for all supplied names.
func (s *SQLiteStore) UpsertQueueStreaks(ctx context.Context, names []string) (map[string]int, error) {
	if len(names) == 0 {
		return nil, nil
	}
	today := time.Now().UTC().Format("2006-01-02")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO queue_streak (name, streak, last_date) VALUES (?, 1, ?)
		ON CONFLICT(name) DO UPDATE SET
			streak    = CASE WHEN julianday(?) - julianday(last_date) <= 4 THEN streak + 1 ELSE 1 END,
			last_date = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, n := range names {
		if _, err = stmt.ExecContext(ctx, n, today, today, today); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	placeholders := make([]string, len(names))
	args := make([]any, len(names))
	for i, n := range names {
		placeholders[i] = "?"
		args[i] = n
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, streak FROM queue_streak WHERE name IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int, len(names))
	for rows.Next() {
		var n string
		var streak int
		if err = rows.Scan(&n, &streak); err != nil {
			return nil, err
		}
		out[n] = streak
	}
	return out, rows.Err()
}

// RegisterSymbols inserts symbols into the registry and returns names that are new (first-ever appearance).
// On the very first call (empty registry), returns nil so the entire symbol universe isn't flagged as IPOs.
func (s *SQLiteStore) RegisterSymbols(ctx context.Context, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	today := time.Now().UTC().Format("2006-01-02")

	// First-run guard: if registry is empty, seed it silently.
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM symbol_registry`).Scan(&count); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO symbol_registry (name, first_seen_date) VALUES (?, ?)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, n := range names {
		if _, err = stmt.ExecContext(ctx, n, today); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil // first run: don't flag everything as IPO
	}

	// Find which names were newly inserted by querying what didn't exist before.
	// Build IN clause for the batch.
	placeholders := make([]string, len(names))
	args := make([]any, len(names))
	for i, n := range names {
		placeholders[i] = "?"
		args[i] = n
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT name FROM symbol_registry WHERE name IN (`+strings.Join(placeholders, ",")+`) AND first_seen_date = ?`,
		append(args, today)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var newNames []string
	for rows.Next() {
		var n string
		if err = rows.Scan(&n); err != nil {
			return nil, err
		}
		newNames = append(newNames, n)
	}
	return newNames, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
