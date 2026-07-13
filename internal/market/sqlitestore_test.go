package market_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"ahrm/internal/indicators"
	"ahrm/internal/market"
)

func TestSQLiteStoreExistingSnapshotDays(t *testing.T) {
	store, err := market.NewSQLiteStore(filepath.Join(t.TempDir(), "market.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	date := "2026-07-13"
	if err := store.UpsertSymbolSnapshot(ctx, date, []indicators.SymbolRow{
		{Name: "فملی", ChangePct: 2.1, Status: "positive"},
		{Name: "خساپا", ChangePct: -1.2, Status: "negative"},
	}); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	got, err := store.ExistingSnapshotDays(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[date]; !ok {
		t.Fatalf("expected snapshot date %s, got %v", date, got)
	}
}

func TestSQLiteStoreSymbolHaltsRoundTrip(t *testing.T) {
	store, err := market.NewSQLiteStore(filepath.Join(t.TempDir(), "market.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	checkedAt := time.Date(2026, 7, 1, 10, 15, 0, 0, time.UTC)
	halts := []market.SymbolHalt{
		{
			Name:                "خساپا",
			Status:              "halted",
			HaltCategory:        "افشای اطلاعات با اهمیت",
			HaltReason:          "به علت افشای اطلاعات با اهمیت",
			HaltedAt:            "1405/04/10 09:05",
			SupervisorMessage:   "نماد متوقف شد",
			SupervisorMessageAt: "1405/04/10 09:05",
		},
	}
	if err := store.ReplaceSymbolHalts(context.Background(), checkedAt, halts); err != nil {
		t.Fatal(err)
	}

	gotCheckedAt, gotHalts, err := store.LatestSymbolHalts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotCheckedAt.IsZero() {
		t.Fatal("checked_at must be stored")
	}
	if len(gotHalts) != 1 || gotHalts[0].Name != "خساپا" {
		t.Fatalf("halts=%+v", gotHalts)
	}
}

func TestSQLiteStoreSymbolHaltEventsRoundTrip(t *testing.T) {
	store, err := market.NewSQLiteStore(filepath.Join(t.TempDir(), "market.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	events := []market.SymbolHaltEvent{
		{
			Symbol:     "خساپا",
			EventType:  "halt",
			Reason:     "افشای اطلاعات با اهمیت",
			OccurredAt: "1405/04/10 09:05",
			Source:     "sourcearena",
			RawMessage: "نماد متوقف شد",
		},
		{
			Symbol:     "خساپا",
			EventType:  "reopen",
			Reason:     "آغاز بازگشایی",
			OccurredAt: "1405/04/10 10:05",
			Source:     "sourcearena",
			RawMessage: "آغاز بازگشایی نماد",
		},
	}
	if err := store.AppendSymbolHaltEvents(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	// Duplicate insert should be ignored (same hash key).
	if err := store.AppendSymbolHaltEvents(context.Background(), events[:1]); err != nil {
		t.Fatal(err)
	}

	got, err := store.RecentSymbolHaltEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("events=%+v", got)
	}
}
