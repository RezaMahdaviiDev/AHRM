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
		if r.Header.Get("X-Header-Token") != "test-token" {
			t.Fatalf("missing header token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "test-token"}, srv.URL, sourcearena.NopRawStore{})
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

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "bad"}, srv.URL, sourcearena.NopRawStore{})
	_, err := client.FetchOptions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeOptionsStringNumbers(t *testing.T) {
	payload := `[{"name":"ضهرم3023","close_price":13500,"close_price_change_percent":"24.56","emal_price":26000,"to_date":"1405/03/27","basis_name":"اهرم","op":"90068"}]`
	opts, err := sourcearena.DecodeOptionsForTest([]byte(payload))
	if err != nil {
		t.Fatalf("DecodeOptionsForTest() error = %v", err)
	}
	if len(opts) != 1 || opts[0].OpenPosition != 90068 {
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
		_, _ = w.Write([]byte(`[{"c":100,"h":110,"l":90,"o":95,"v":10,"t":1},{"c":105,"h":112,"l":92,"o":100,"v":12,"t":2}]`))
	}))
	defer srv.Close()

	client := sourcearena.NewTestClient(config.SourceArenaConfig{APIToken: "test-token"}, srv.URL, sourcearena.NopRawStore{})
	candles, err := client.FetchDailyCandles(context.Background(), "اهرم", time.Unix(1, 0), time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 2 {
		t.Fatalf("len(candles)=%d", len(candles))
	}
}
