package market_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"ahrm/internal/market"
)

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
