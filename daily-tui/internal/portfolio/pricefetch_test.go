package portfolio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseYahooChartHappyPath(t *testing.T) {
	body := `{"chart":{"result":[{"meta":{"regularMarketPrice":207.15,"symbol":"AAPL"}}],"error":null}}`
	price, err := parseYahooChart(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if price != 207.15 {
		t.Errorf("price = %v, want 207.15", price)
	}
}

func TestParseYahooChartError(t *testing.T) {
	cases := map[string]string{
		"yahoo error":  `{"chart":{"result":[],"error":{"code":"Not Found","description":"no data"}}}`,
		"empty result": `{"chart":{"result":[],"error":null}}`,
		"zero price":   `{"chart":{"result":[{"meta":{"regularMarketPrice":0}}],"error":null}}`,
		"bad json":     `not json`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseYahooChart(strings.NewReader(body)); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestYahooFetcherHitsEndpoint(t *testing.T) {
	var gotPath, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"regularMarketPrice":182.4}}]}}`))
	}))
	defer srv.Close()

	f := &YahooFetcher{Client: srv.Client(), BaseURL: srv.URL + "/"}
	price, err := f.Fetch("AAPL")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if price != 182.4 {
		t.Errorf("price = %v, want 182.4", price)
	}
	if gotPath != "/AAPL" {
		t.Errorf("path = %q, want /AAPL", gotPath)
	}
	if gotUA == "" {
		t.Error("expected User-Agent header to be set")
	}
}

func TestYahooFetcherNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	f := &YahooFetcher{Client: srv.Client(), BaseURL: srv.URL + "/"}
	if _, err := f.Fetch("ZZZZ"); err == nil {
		t.Error("expected error on 404")
	}
}

func TestYahooFetcherEmptyTicker(t *testing.T) {
	f := NewYahooFetcher()
	if _, err := f.Fetch(""); err == nil {
		t.Error("expected error for empty ticker")
	}
}
