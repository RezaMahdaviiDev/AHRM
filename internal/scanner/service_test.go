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

	"ahrm/internal/config"
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
		case q.Has("all"):
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

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "t"}, srv.URL, sourcearena.NopRawStore{})
	cfg := &config.Config{}
	svc := scanner.NewService(cfg, client, nil, nil)
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
	b, _ := json.Marshal(snap.Opportunities[0])
	if len(b) == 0 {
		t.Fatal("empty opp")
	}
}
