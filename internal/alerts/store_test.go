package alerts_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"testing"

	"ahrm/internal/alerts"
)

func TestSQLiteStorePrunesStaleRowsOnStartup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "alerts.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS alert_history (
		alert_type TEXT NOT NULL,
		alert_key  TEXT NOT NULL,
		payload    TEXT NOT NULL DEFAULT '{}',
		sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (alert_type, alert_key)
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO alert_history (alert_type, alert_key, payload, sent_at) VALUES (?, ?, ?, datetime('now', '-25 hours'))`,
		"matrix", digestFor("matrix", "rule-old"), "{}",
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := alerts.NewSQLiteStore(dbPath); err != nil {
		t.Fatal(err)
	}

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM alert_history`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected stale rows pruned on startup, count=%d", count)
	}
}

func TestSQLiteStoreRecordRefreshesStaleEntry(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "alerts.db")

	store, err := alerts.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	alertType := "matrix"
	key := "rule-1"
	digest := digestFor(alertType, key)

	if _, err := db.Exec(
		`INSERT INTO alert_history (alert_type, alert_key, payload, sent_at) VALUES (?, ?, ?, datetime('now', '-25 hours'))`,
		alertType, digest, `{"message":"old"}`,
	); err != nil {
		t.Fatal(err)
	}

	sent, err := store.WasSent(ctx, alertType, key)
	if err != nil {
		t.Fatal(err)
	}
	if sent {
		t.Fatal("expected stale row ignored by WasSent")
	}

	if err := store.Record(ctx, alertType, key, []byte(`{"message":"new"}`)); err != nil {
		t.Fatal(err)
	}

	sent, err = store.WasSent(ctx, alertType, key)
	if err != nil {
		t.Fatal(err)
	}
	if !sent {
		t.Fatal("expected record refresh to make alert sent within 24h")
	}

	var recent int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM alert_history WHERE alert_type = ? AND alert_key = ? AND sent_at > datetime('now', '-24 hours')`,
		alertType, digest,
	).Scan(&recent); err != nil {
		t.Fatal(err)
	}
	if recent != 1 {
		t.Fatalf("expected exactly one recent row, got %d", recent)
	}

	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM alert_history WHERE alert_type = ? AND alert_key = ?`,
		alertType, digest,
	).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("expected one row after replace, got %d", total)
	}
}

func digestFor(alertType, key string) string {
	sum := sha256.Sum256([]byte(alertType + ":" + key))
	return hex.EncodeToString(sum[:])
}
