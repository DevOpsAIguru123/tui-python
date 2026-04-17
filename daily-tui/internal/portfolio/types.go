package portfolio

// Holding is a single stock position the user owns.
// LastPrice is optional — a zero value means the user has not entered a
// current price and derived metrics (market value, P/L) must render as
// unknown rather than $0.
type Holding struct {
	Ticker    string  `json:"ticker"`
	Shares    float64 `json:"shares"`
	AvgCost   float64 `json:"avg_cost"`
	LastPrice float64 `json:"last_price,omitempty"`
	Note      string  `json:"note,omitempty"`
}

// HasPrice reports whether LastPrice is populated (> 0).
func (h Holding) HasPrice() bool { return h.LastPrice > 0 }

// CostBasis is shares × avg cost.
func (h Holding) CostBasis() float64 { return h.Shares * h.AvgCost }

// MarketValue is shares × last price. Only meaningful when HasPrice is true.
func (h Holding) MarketValue() float64 { return h.Shares * h.LastPrice }

// PL is unrealised profit/loss in dollars.
// Only meaningful when HasPrice is true.
func (h Holding) PL() float64 { return h.MarketValue() - h.CostBasis() }

// PLPct is unrealised P/L as a percentage of cost basis.
// Returns 0 when cost basis is zero to avoid divide-by-zero.
func (h Holding) PLPct() float64 {
	cb := h.CostBasis()
	if cb == 0 {
		return 0
	}
	return (h.PL() / cb) * 100
}

// WatchItem is a ticker the user wants to keep an eye on, with no position.
type WatchItem struct {
	Ticker string `json:"ticker"`
	Note   string `json:"note,omitempty"`
}
