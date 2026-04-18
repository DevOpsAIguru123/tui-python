package portfolio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Fetcher resolves the current market price for a ticker.
// Implementations should be safe for sequential use from a single goroutine;
// the CLI calls Fetch once per holding.
type Fetcher interface {
	Fetch(ticker string) (float64, error)
}

// YahooFetcher pulls the latest regular-market price from Yahoo Finance's
// public chart endpoint. No API key required. The endpoint is unofficial
// but widely relied on — if Yahoo changes it, this file is the only thing
// that needs to change.
type YahooFetcher struct {
	Client  *http.Client
	BaseURL string // default: https://query1.finance.yahoo.com/v8/finance/chart/
}

// NewYahooFetcher returns a YahooFetcher with sensible defaults.
func NewYahooFetcher() *YahooFetcher {
	return &YahooFetcher{
		Client:  &http.Client{Timeout: 8 * time.Second},
		BaseURL: "https://query1.finance.yahoo.com/v8/finance/chart/",
	}
}

// Fetch returns the most recent regular-market price for ticker.
// Returns an error if the ticker is unknown, the response is malformed,
// or the endpoint is unreachable.
func (y *YahooFetcher) Fetch(ticker string) (float64, error) {
	if ticker == "" {
		return 0, fmt.Errorf("ticker is required")
	}
	u := y.BaseURL + url.PathEscape(ticker)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	// Yahoo rejects requests without a browser-ish User-Agent.
	req.Header.Set("User-Agent", "daily-tui/1.0 (+https://github.com/DevOpsAIguru123/productivity-tools)")
	resp, err := y.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("yahoo %s: %s: %s", ticker, resp.Status, string(body))
	}
	return parseYahooChart(resp.Body)
}

// yahooChartResponse is the minimal subset of Yahoo's chart payload we need.
type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				Symbol             string  `json:"symbol"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func parseYahooChart(r io.Reader) (float64, error) {
	var payload yahooChartResponse
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode yahoo response: %w", err)
	}
	if payload.Chart.Error != nil {
		return 0, fmt.Errorf("yahoo: %s: %s", payload.Chart.Error.Code, payload.Chart.Error.Description)
	}
	if len(payload.Chart.Result) == 0 {
		return 0, fmt.Errorf("yahoo: no result for ticker")
	}
	price := payload.Chart.Result[0].Meta.RegularMarketPrice
	if price <= 0 {
		return 0, fmt.Errorf("yahoo: no price in response")
	}
	return price, nil
}
