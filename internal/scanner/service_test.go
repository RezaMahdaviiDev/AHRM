package scanner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ahrm/internal/alerts"
	"ahrm/internal/config"
	"ahrm/internal/indicators"
	"ahrm/internal/market"
	"ahrm/internal/scanner"
	"ahrm/internal/sourcearena"
)

func TestRefreshWithMockAPI(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "sourcearena", "testdata", "options.json"))
	if err != nil {
		t.Fatal(err)
	}
	symbols := `[{"name":"اهرم","close_price":25000,"close_price_change_percent":1.2},{"name":"فملی","close_price":1000,"close_price_change_percent":-1}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case q.Get("all") == "e":
			_, _ = w.Write(raw)
		case q.Get("type") != "":
			_, _ = w.Write([]byte(symbols))
		case q.Get("name") != "":
			_, _ = w.Write([]byte(`{"name":"اهرم","close_price":25000,"close_price_change_percent":1.2}`))
		default:
			candles := make([]byte, 0, 4096)
			candles = append(candles, '[')
			for i := 0; i < 45; i++ {
				if i > 0 {
					candles = append(candles, ',')
				}
				candles = append(candles, []byte(fmt.Sprintf(`{"c":%d}`, 24000+i*20))...)
			}
			candles = append(candles, ']')
			_, _ = w.Write(candles)
		}
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "t"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	cfg := &config.Config{}
	svc := scanner.NewService(cfg, client, nil, nil, nil, nil)
	snap, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Underlying.ClosePrice != 25000 {
		t.Fatalf("underlying=%v", snap.Underlying.ClosePrice)
	}
	if len(snap.Opportunities) == 0 {
		t.Fatalf("expected opportunities from fixture pair")
	}
	if len(snap.CallMatrices) < 1 {
		t.Fatalf("expected call matrices from fixture, got %d", len(snap.CallMatrices))
	}
	if len(snap.PutMatrices) < 1 {
		t.Fatalf("expected put matrices from fixture, got %d", len(snap.PutMatrices))
	}
	b, _ := json.Marshal(snap.Opportunities[0])
	if len(b) == 0 {
		t.Fatal("empty opp")
	}
	if snap.ImpliedVolatility == nil {
		t.Fatal("ImpliedVolatility should be populated")
	}
}

type holidayDailyStore struct {
	history []indicators.DailyMarket
}

func (s *holidayDailyStore) UpsertToday(context.Context, indicators.DailyMarket) error { return nil }
func (s *holidayDailyStore) UpsertDay(context.Context, time.Time, indicators.DailyMarket) error {
	return nil
}
func (s *holidayDailyStore) LastDays(context.Context, int) ([]indicators.DailyMarket, error) {
	return s.history, nil
}
func (s *holidayDailyStore) ExistingDays(context.Context, time.Time, time.Time) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

type captureSender struct {
	messages []string
}

func (c *captureSender) SendMessage(_ context.Context, text string) error {
	c.messages = append(c.messages, text)
	return nil
}

var _ market.DailyStore = (*holidayDailyStore)(nil)

func TestRefreshSkipsAlertsOnHolidaySnapshot(t *testing.T) {
	rawOptions := `[
		{"name":"ضهرم9000","close_price":500,"emal_price":32000,"to_date":"1406/12/29","trade_volume":10},
		{"name":"طهرم9000","close_price":200,"emal_price":32000,"to_date":"1406/12/29","trade_volume":8}
	]`
	// Identical breadth stats to last stored day => holiday/cached snapshot.
	symbols := `[
		{"name":"فملی","market":"بازار اول (تابلوی اصلی) بورس","close_price":1000,"final_price_change_percent":1.1,"trade_value":100},
		{"name":"فولاد","market":"بازار اول (تابلوی اصلی) بورس","close_price":900,"final_price_change_percent":-1.2,"trade_value":100}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case q.Get("all") == "e":
			_, _ = w.Write([]byte(rawOptions))
		case q.Get("type") != "":
			_, _ = w.Write([]byte(symbols))
		case q.Get("name") != "":
			_, _ = w.Write([]byte(`{"name":"اهرم","close_price":25000}`))
		default:
			candles := make([]byte, 0, 4096)
			candles = append(candles, '[')
			for i := 0; i < 45; i++ {
				if i > 0 {
					candles = append(candles, ',')
				}
				candles = append(candles, []byte(fmt.Sprintf(`{"c":%d}`, 24000+i*20))...)
			}
			candles = append(candles, ']')
			_, _ = w.Write(candles)
		}
	}))
	defer srv.Close()

	sender := &captureSender{}
	alertEngine := alerts.NewEngine(alerts.Config{
		ArbitrageR12Threshold:   0.01,
		CoveredCallROIThreshold: 0.01,
		BullSpreadATMThreshold:  0.01,
		BullSpreadOTMThreshold:  0.01,
	}, sender, alerts.NewMemStore())

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "t"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	cfg := &config.Config{
		Alerts: config.AlertsConfig{
			BreadthHighThreshold: 0.6,
			BreadthLowThreshold:  0.4,
			AdvanceHighThreshold: 1.4,
			AdvanceLowThreshold:  0.6,
		},
	}
	store := &holidayDailyStore{
		history: []indicators.DailyMarket{{Date: "2026-07-02", Positive: 1, Negative: 1, Total: 2}},
	}
	svc := scanner.NewService(cfg, client, store, nil, nil, alertEngine)

	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(sender.messages); got != 0 {
		t.Fatalf("expected no alerts on stale holiday snapshot, got %d", got)
	}
}
