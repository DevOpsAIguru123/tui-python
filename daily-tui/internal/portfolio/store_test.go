package portfolio

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "portfolio.json"))
}

func TestStore_roundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")

	s1 := NewStore(path)
	s1.AddHolding(Holding{Ticker: "AAPL", Shares: 25, AvgCost: 182.40, LastPrice: 207.15, Note: "long-term"})
	s1.AddWatch(WatchItem{Ticker: "NVDA", Note: "earnings soon"})

	s2 := NewStore(path)
	if got := s2.Holdings(); len(got) != 1 || got[0].Ticker != "AAPL" {
		t.Errorf("holdings not persisted: %+v", got)
	}
	if got := s2.Watchlist(); len(got) != 1 || got[0].Ticker != "NVDA" {
		t.Errorf("watchlist not persisted: %+v", got)
	}
}

func TestStore_updateHolding(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100})
	s.UpdateHolding(0, Holding{Ticker: "AAPL", Shares: 20, AvgCost: 110})

	got := s.Holdings()
	if got[0].Shares != 20 || got[0].AvgCost != 110 {
		t.Errorf("update did not take effect: %+v", got[0])
	}
}

func TestStore_deleteHolding(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL"})
	s.AddHolding(Holding{Ticker: "MSFT"})
	s.DeleteHolding(0)

	got := s.Holdings()
	if len(got) != 1 || got[0].Ticker != "MSFT" {
		t.Errorf("delete left wrong state: %+v", got)
	}
}

func TestStore_setLastPrice(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100})
	s.SetLastPrice(0, 125.50)
	if got := s.Holdings()[0].LastPrice; got != 125.50 {
		t.Errorf("SetLastPrice: got %v, want 125.50", got)
	}
}

func TestStore_outOfRangeIgnored(t *testing.T) {
	s := newTestStore(t)
	// Should not panic or corrupt the store.
	s.UpdateHolding(99, Holding{Ticker: "X"})
	s.DeleteHolding(99)
	s.SetLastPrice(99, 1.0)
	s.UpdateWatch(99, WatchItem{Ticker: "X"})
	s.DeleteWatch(99)
	if len(s.Holdings()) != 0 || len(s.Watchlist()) != 0 {
		t.Error("out-of-range operations mutated store")
	}
}

func TestStore_watchCRUD(t *testing.T) {
	s := newTestStore(t)
	s.AddWatch(WatchItem{Ticker: "NVDA"})
	s.AddWatch(WatchItem{Ticker: "TSLA"})
	s.UpdateWatch(1, WatchItem{Ticker: "TSLA", Note: "earnings"})
	s.DeleteWatch(0)

	got := s.Watchlist()
	if len(got) != 1 || got[0].Ticker != "TSLA" || got[0].Note != "earnings" {
		t.Errorf("watch CRUD wrong state: %+v", got)
	}
}

func TestStore_holdingsReturnsCopy(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL"})

	out := s.Holdings()
	out[0].Ticker = "MUTATED"

	if s.Holdings()[0].Ticker != "AAPL" {
		t.Error("Holdings() should return a copy; internal state was mutated")
	}
}

func TestStore_totals(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100, LastPrice: 120})
	s.AddHolding(Holding{Ticker: "MSFT", Shares: 5, AvgCost: 200, LastPrice: 180})
	s.AddHolding(Holding{Ticker: "TSLA", Shares: 2, AvgCost: 250}) // no price

	cb, mv, pl, hasAny := s.Totals()
	wantCB := 10*100 + 5*200 + 2*250              // 2500
	wantMV := 10*120 + 5*180                      // 2100
	wantPL := (10*120 - 10*100) + (5*180 - 5*200) // 200 - 100 = 100

	if cb != float64(wantCB) {
		t.Errorf("cost basis: got %v, want %v", cb, wantCB)
	}
	if mv != float64(wantMV) {
		t.Errorf("market value: got %v, want %v", mv, wantMV)
	}
	if pl != float64(wantPL) {
		t.Errorf("P/L: got %v, want %v", pl, wantPL)
	}
	if !hasAny {
		t.Error("expected hasAnyPrice true")
	}
}

func TestStore_totals_noPrices(t *testing.T) {
	s := newTestStore(t)
	s.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100})
	_, mv, pl, hasAny := s.Totals()
	if hasAny {
		t.Error("expected hasAnyPrice false when no holding has a price")
	}
	if mv != 0 || pl != 0 {
		t.Errorf("expected MV/PL zero when no prices; got %v / %v", mv, pl)
	}
}

func TestHolding_calculations(t *testing.T) {
	h := Holding{Shares: 10, AvgCost: 100, LastPrice: 120}
	if got := h.CostBasis(); got != 1000 {
		t.Errorf("CostBasis: %v", got)
	}
	if got := h.MarketValue(); got != 1200 {
		t.Errorf("MarketValue: %v", got)
	}
	if got := h.PL(); got != 200 {
		t.Errorf("PL: %v", got)
	}
	if got := h.PLPct(); got != 20 {
		t.Errorf("PLPct: got %v, want 20", got)
	}
}

func TestHolding_PLPctZeroCost(t *testing.T) {
	h := Holding{Shares: 10, AvgCost: 0, LastPrice: 120}
	if got := h.PLPct(); got != 0 {
		t.Errorf("PLPct with zero cost should be 0, got %v", got)
	}
}

func TestHolding_HasPrice(t *testing.T) {
	if (Holding{LastPrice: 100}).HasPrice() == false {
		t.Error("HasPrice should be true for non-zero LastPrice")
	}
	if (Holding{}).HasPrice() {
		t.Error("HasPrice should be false for zero LastPrice")
	}
}
