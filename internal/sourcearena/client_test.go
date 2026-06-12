package sourcearena_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ahrm/internal/config"
	"ahrm/internal/sourcearena"
)

func TestDecodeOptionsFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "options.json"))
	if err != nil {
		t.Fatal(err)
	}
	opts, err := sourcearena.DecodeOptionsForTest(raw)
	if err != nil {
		t.Fatalf("DecodeOptionsForTest() error = %v", err)
	}
	if len(opts) != 3 {
		t.Fatalf("len(options) = %d, want 3", len(opts))
	}
}

func TestClientFetchOptions(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "options.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Header-Token") != "" {
			t.Fatalf("market API must not use X-Header-Token header")
		}
		if r.URL.Query().Get("all") != "e" {
			t.Fatalf("unexpected query: %v", r.URL.Query())
		}
		if r.URL.Query().Get("token") != "test-token" {
			t.Fatalf("token query param required for market API")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "test-token"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	opts, err := client.FetchOptions(context.Background())
	if err != nil {
		t.Fatalf("FetchOptions() error = %v", err)
	}
	if len(opts) != 3 {
		t.Fatalf("len(options) = %d, want 3", len(opts))
	}
}

func TestClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"Error":"invalid token"}`))
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "bad"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	_, err := client.FetchOptions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeOptionsStringNumbers(t *testing.T) {
	payload := `[{"name":"ضهرم3023","close_price":13500,"close_price_change_percent":"24.56","emal_price":26000,"to_date":"1405/03/27","basis_name":"اهرم","op":"90068","1_sell_volume":"7152","1_buy_volume":"32640691","trade_volume":"12345"}]`
	opts, err := sourcearena.DecodeOptionsForTest([]byte(payload))
	if err != nil {
		t.Fatalf("DecodeOptionsForTest() error = %v", err)
	}
	if len(opts) != 1 || opts[0].OpenPosition != 90068 || opts[0].SellRow1Volume != 7152 || opts[0].BuyRow1Volume != 32640691 || opts[0].TradeVolume != 12345 {
		t.Fatalf("got %+v", opts)
	}
}

func TestDecodeSymbolsStringNumbers(t *testing.T) {
	payload := `{"name":"اهرم","close_price":"37649","close_price_change_percent":"4%"}`
	symbols, err := sourcearena.DecodeSymbolsForTest(json.RawMessage(payload))
	if err != nil {
		t.Fatal(err)
	}
	if symbols[0].ClosePrice != 37649 || symbols[0].ClosePriceChangePct != 4 {
		t.Fatalf("got %+v", symbols[0])
	}
}

func TestDecodeSymbolsArray(t *testing.T) {
	payload := `[{"name":"اهرم","close_price":25000,"close_price_change_percent":1.2}]`
	var raw json.RawMessage = []byte(payload)
	symbols, err := sourcearena.DecodeSymbolsForTest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if symbols[0].Name != "اهرم" {
		t.Fatalf("name = %q", symbols[0].Name)
	}
}

func TestFetchDailyCandles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Header-Token"); got != "test-token" {
			t.Fatalf("X-Header-Token=%q, want test-token", got)
		}
		if r.URL.Query().Get("token") != "" {
			t.Fatalf("candle API must not use token query param")
		}
		q := r.URL.Query()
		if q.Get("symbol") != "اهرم" || q.Get("resolution") != "1D" || q.Get("type") != "1" {
			t.Fatalf("unexpected query: %v", q)
		}
		_, _ = w.Write([]byte(`[{"c":100,"h":110,"l":90,"o":95,"v":10,"t":1},{"c":105,"h":112,"l":92,"o":100,"v":12,"t":2}]`))
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "test-token"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	candles, err := client.FetchDailyCandles(context.Background(), "اهرم", time.Unix(1, 0), time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 {
		t.Fatalf("len(candles)=%d", len(candles))
	}
}

func TestDecodeCandlesAPIError(t *testing.T) {
	_, err := sourcearena.DecodeCandlesForTest([]byte(`{"success":false,"message":"توکن وب سرویس را وارد کنید","error_code":1001}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeCandlesWrappedData(t *testing.T) {
	candles, err := sourcearena.DecodeCandlesForTest([]byte(`{"success":true,"data":[{"c":100,"t":1},{"c":105,"t":2}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 {
		t.Fatalf("len=%d", len(candles))
	}
}

func TestFetchCandlesCustomType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "4" || r.URL.Query().Get("resolution") != "1W" {
			t.Fatalf("query=%v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`[{"c":100,"t":1}]`))
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "test-token"}, srv.URL, srv.URL, sourcearena.NopRawStore{})
	_, err := client.FetchCandles(context.Background(), sourcearena.CandleRequest{
		Symbol: "فملی", From: time.Unix(1, 0), To: time.Unix(2, 0),
		Resolution: "1W", Type: sourcearena.AdjustPerformance,
	})
	if err != nil {
		t.Fatal(err)
	}
}
