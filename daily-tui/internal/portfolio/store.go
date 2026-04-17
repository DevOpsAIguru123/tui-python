package portfolio

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Data is the on-disk shape of the portfolio file.
type Data struct {
	Holdings  []Holding   `json:"holdings"`
	Watchlist []WatchItem `json:"watchlist"`
}

// Store persists Data to a JSON file. Every mutating method saves
// synchronously — the file is small enough that the cost is irrelevant
// and it removes a whole class of "forgot to save" bugs.
type Store struct {
	path string
	data Data
}

// NewStore loads portfolio data from path (returns empty store if absent).
func NewStore(path string) *Store {
	s := &Store{path: path}
	s.load()
	return s
}

func (s *Store) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &s.data)
}

func (s *Store) save() {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	_ = os.WriteFile(s.path, raw, 0644)
}

// Holdings returns a copy of the current holdings list.
func (s *Store) Holdings() []Holding {
	out := make([]Holding, len(s.data.Holdings))
	copy(out, s.data.Holdings)
	return out
}

// Watchlist returns a copy of the current watchlist.
func (s *Store) Watchlist() []WatchItem {
	out := make([]WatchItem, len(s.data.Watchlist))
	copy(out, s.data.Watchlist)
	return out
}

// AddHolding appends h and persists.
func (s *Store) AddHolding(h Holding) {
	s.data.Holdings = append(s.data.Holdings, h)
	s.save()
}

// UpdateHolding replaces the holding at index i with h and persists.
// Out-of-range indexes are silently ignored (same convention as slices).
func (s *Store) UpdateHolding(i int, h Holding) {
	if i < 0 || i >= len(s.data.Holdings) {
		return
	}
	s.data.Holdings[i] = h
	s.save()
}

// DeleteHolding removes the holding at index i and persists.
func (s *Store) DeleteHolding(i int) {
	if i < 0 || i >= len(s.data.Holdings) {
		return
	}
	s.data.Holdings = append(s.data.Holdings[:i], s.data.Holdings[i+1:]...)
	s.save()
}

// SetLastPrice updates only the LastPrice field of holding i.
func (s *Store) SetLastPrice(i int, price float64) {
	if i < 0 || i >= len(s.data.Holdings) {
		return
	}
	s.data.Holdings[i].LastPrice = price
	s.save()
}

// AddWatch appends w and persists.
func (s *Store) AddWatch(w WatchItem) {
	s.data.Watchlist = append(s.data.Watchlist, w)
	s.save()
}

// UpdateWatch replaces the watch entry at i with w and persists.
func (s *Store) UpdateWatch(i int, w WatchItem) {
	if i < 0 || i >= len(s.data.Watchlist) {
		return
	}
	s.data.Watchlist[i] = w
	s.save()
}

// DeleteWatch removes the watch entry at index i and persists.
func (s *Store) DeleteWatch(i int) {
	if i < 0 || i >= len(s.data.Watchlist) {
		return
	}
	s.data.Watchlist = append(s.data.Watchlist[:i], s.data.Watchlist[i+1:]...)
	s.save()
}

// Totals returns aggregate cost basis, market value (for rows with prices),
// and realised P/L across all holdings. hasAnyPrice reports whether any
// holding has a LastPrice set; if false the UI hides market-value/P/L
// totals rather than showing a misleading value.
func (s *Store) Totals() (costBasis, marketValue, pl float64, hasAnyPrice bool) {
	for _, h := range s.data.Holdings {
		costBasis += h.CostBasis()
		if h.HasPrice() {
			marketValue += h.MarketValue()
			pl += h.PL()
			hasAnyPrice = true
		}
	}
	return
}
