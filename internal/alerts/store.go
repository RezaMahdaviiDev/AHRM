package alerts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"
)

// AlertStore persists sent-alert history to prevent duplicate notifications across restarts.
type AlertStore interface {
	WasSent(ctx context.Context, alertType, key string) (bool, error)
	Record(ctx context.Context, alertType, key string, payload []byte) error
}

// PostgresStore persists alert history in PostgreSQL (used when Supabase is configured).
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) WasSent(ctx context.Context, alertType, key string) (bool, error) {
	digest := hashKey(alertType + ":" + key)
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM alert_history WHERE alert_type=$1 AND alert_key=$2)`,
		alertType, digest,
	).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) Record(ctx context.Context, alertType, key string, payload []byte) error {
	digest := hashKey(alertType + ":" + key)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO alert_history (alert_type, alert_key, payload, sent_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (alert_type, alert_key) DO NOTHING`,
		alertType, digest, payload, time.Now().UTC(),
	)
	return err
}

// SQLiteStore persists alert history in a local SQLite file.
// Used when PostgreSQL is not configured, so notifications survive server restarts.
// Alerts older than 24 hours are pruned on startup so daily opportunities can re-trigger.
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS alert_history (
		alert_type TEXT NOT NULL,
		alert_key  TEXT NOT NULL,
		payload    TEXT NOT NULL DEFAULT '{}',
		sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (alert_type, alert_key)
	)`); err != nil {
		db.Close()
		return nil, err
	}
	// Prune entries older than 24 hours so daily opportunities re-trigger after a new market session.
	if _, err = db.Exec(`DELETE FROM alert_history WHERE sent_at < datetime('now', '-24 hours')`); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) WasSent(_ context.Context, alertType, key string) (bool, error) {
	digest := hashKey(alertType + ":" + key)
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM alert_history WHERE alert_type = ? AND alert_key = ?`,
		alertType, digest,
	).Scan(&count)
	return count > 0, err
}

func (s *SQLiteStore) Record(_ context.Context, alertType, key string, payload []byte) error {
	digest := hashKey(alertType + ":" + key)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO alert_history (alert_type, alert_key, payload, sent_at) VALUES (?, ?, ?, ?)`,
		alertType, digest, string(payload), time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

// NewMemStore returns an in-memory AlertStore with no persistence.
// Intended for tests only.
func NewMemStore() AlertStore {
	s, _ := NewSQLiteStore(":memory:")
	return s
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
