package portfolio

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "portfolio.json"))
	return New(store)
}

// typeRunes simulates the user typing literal characters into the input.
// The textinput widget handles KeyRunes one at a time via Update.
func typeRunes(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	return m
}

func pressEnter(t *testing.T, m Model) Model {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(Model)
}

func pressKey(t *testing.T, m Model, r rune) Model {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return next.(Model)
}

func TestInputting_idleByDefault(t *testing.T) {
	m := newTestModel(t)
	if m.Inputting() {
		t.Error("new model should not be inputting")
	}
}

func TestSectionSwitch(t *testing.T) {
	m := newTestModel(t)
	if m.Section() != SectionHoldings {
		t.Errorf("default section = %v, want SectionHoldings", m.Section())
	}
	m = pressKey(t, m, '2')
	if m.Section() != SectionWatchlist {
		t.Errorf("after '2': %v, want SectionWatchlist", m.Section())
	}
	m = pressKey(t, m, '1')
	if m.Section() != SectionHoldings {
		t.Errorf("after '1': %v, want SectionHoldings", m.Section())
	}
}

func TestAddHolding_happyPath(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, 'a')
	if !m.Inputting() {
		t.Fatal("pressing 'a' should open the add form")
	}

	m = typeRunes(t, m, "aapl")
	m = pressEnter(t, m) // ticker
	m = typeRunes(t, m, "25")
	m = pressEnter(t, m) // shares
	m = typeRunes(t, m, "182.40")
	m = pressEnter(t, m) // avg cost
	m = typeRunes(t, m, "207.15")
	m = pressEnter(t, m) // last price
	m = typeRunes(t, m, "long-term")
	m = pressEnter(t, m) // note

	if m.Inputting() {
		t.Fatal("form should close after final field")
	}
	holdings := m.store.Holdings()
	if len(holdings) != 1 {
		t.Fatalf("want 1 holding, got %d", len(holdings))
	}
	got := holdings[0]
	if got.Ticker != "AAPL" {
		t.Errorf("ticker should be uppercased: %q", got.Ticker)
	}
	if got.Shares != 25 || got.AvgCost != 182.40 || got.LastPrice != 207.15 {
		t.Errorf("numeric fields wrong: %+v", got)
	}
	if got.Note != "long-term" {
		t.Errorf("note: %q", got.Note)
	}
}

func TestAddHolding_blankLastPrice(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, 'a')

	m = typeRunes(t, m, "tsla")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "2")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "250")
	m = pressEnter(t, m)
	m = pressEnter(t, m) // blank last price
	m = pressEnter(t, m) // blank note

	got := m.store.Holdings()[0]
	if got.HasPrice() {
		t.Errorf("blank last price should yield HasPrice()==false, got %+v", got)
	}
}

func TestAddHolding_validationRejectsBadShares(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, 'a')

	m = typeRunes(t, m, "aapl")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "not-a-number")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "100")
	m = pressEnter(t, m)
	m = pressEnter(t, m) // last price blank
	m = pressEnter(t, m) // note blank — triggers finalize

	if !m.Inputting() {
		t.Error("invalid shares should keep the form open for correction")
	}
	if m.err == "" {
		t.Error("expected validation error message")
	}
	if len(m.store.Holdings()) != 0 {
		t.Error("no holding should be persisted on validation failure")
	}
}

func TestAddHolding_missingTickerRejected(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, 'a')

	// Blank ticker → blank everything; finalize should fail.
	m = pressEnter(t, m)
	m = typeRunes(t, m, "1")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "1")
	m = pressEnter(t, m)
	m = pressEnter(t, m)
	m = pressEnter(t, m)

	if !m.Inputting() {
		t.Error("blank ticker should keep form open")
	}
	if len(m.store.Holdings()) != 0 {
		t.Error("no holding should be persisted when ticker is blank")
	}
}

func TestEditHolding_updatesExisting(t *testing.T) {
	m := newTestModel(t)
	m.store.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100})

	m = pressKey(t, m, 'e')
	if !m.Inputting() {
		t.Fatal("edit should open form")
	}
	// Field 0 is pre-populated with "AAPL"; confirm through all fields with edits.
	m = pressEnter(t, m)   // keep ticker
	m.input.SetValue("20") // new shares
	m = pressEnter(t, m)
	m.input.SetValue("110") // new avg cost
	m = pressEnter(t, m)
	m = pressEnter(t, m) // last price unchanged (blank)
	m = pressEnter(t, m) // note unchanged (blank)

	got := m.store.Holdings()
	if len(got) != 1 || got[0].Shares != 20 || got[0].AvgCost != 110 {
		t.Errorf("edit did not persist: %+v", got)
	}
}

func TestDeleteHolding_removesSelected(t *testing.T) {
	m := newTestModel(t)
	m.store.AddHolding(Holding{Ticker: "AAPL"})
	m.store.AddHolding(Holding{Ticker: "MSFT"})

	m = pressKey(t, m, 'd')
	got := m.store.Holdings()
	if len(got) != 1 || got[0].Ticker != "MSFT" {
		t.Errorf("delete wrong result: %+v", got)
	}
}

func TestUpdatePrice_holdingOnly(t *testing.T) {
	m := newTestModel(t)
	m.store.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100})

	m = pressKey(t, m, 'p')
	if !m.Inputting() {
		t.Fatal("'p' should open price-update form")
	}
	m.input.SetValue("125.50")
	m = pressEnter(t, m)

	if m.Inputting() {
		t.Fatal("form should close after price entry")
	}
	if got := m.store.Holdings()[0].LastPrice; got != 125.50 {
		t.Errorf("LastPrice: got %v, want 125.50", got)
	}
}

func TestUpdatePrice_notAvailableInWatchlist(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, '2') // switch to watchlist
	m = pressKey(t, m, 'p')
	if m.Inputting() {
		t.Error("'p' should be a no-op in watchlist section")
	}
}

func TestAddWatch(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, '2')
	m = pressKey(t, m, 'a')

	m = typeRunes(t, m, "nvda")
	m = pressEnter(t, m)
	m = typeRunes(t, m, "earnings next week")
	m = pressEnter(t, m)

	got := m.store.Watchlist()
	if len(got) != 1 || got[0].Ticker != "NVDA" || got[0].Note != "earnings next week" {
		t.Errorf("watch add wrong: %+v", got)
	}
}

func TestEscapeCancelsForm(t *testing.T) {
	m := newTestModel(t)
	m = pressKey(t, m, 'a')
	m = typeRunes(t, m, "aapl")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)

	if m.Inputting() {
		t.Error("Esc should close the form")
	}
	if len(m.store.Holdings()) != 0 {
		t.Error("Esc should not persist anything")
	}
}

func TestCursorNavigation(t *testing.T) {
	m := newTestModel(t)
	m.store.AddHolding(Holding{Ticker: "A"})
	m.store.AddHolding(Holding{Ticker: "B"})

	if m.Cursor() != 0 {
		t.Fatalf("initial cursor %d", m.Cursor())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(Model)
	if m.Cursor() != 1 {
		t.Errorf("after Down: %d", m.Cursor())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(Model)
	if m.Cursor() != 1 {
		t.Errorf("Down past end should clamp at 1, got %d", m.Cursor())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(Model)
	if m.Cursor() != 0 {
		t.Errorf("after Up: %d", m.Cursor())
	}
}

func TestView_emptyStates(t *testing.T) {
	m := newTestModel(t)
	if !strings.Contains(m.View(), "No holdings yet") {
		t.Error("empty holdings should render hint")
	}
	m = pressKey(t, m, '2')
	if !strings.Contains(m.View(), "No watchlist") {
		t.Error("empty watchlist should render hint")
	}
}

func TestView_rendersPLOrDash(t *testing.T) {
	m := newTestModel(t)
	m.store.AddHolding(Holding{Ticker: "AAPL", Shares: 10, AvgCost: 100, LastPrice: 120})
	m.store.AddHolding(Holding{Ticker: "MSFT", Shares: 5, AvgCost: 200}) // no price

	out := m.View()
	if !strings.Contains(out, "AAPL") || !strings.Contains(out, "MSFT") {
		t.Errorf("rows missing: %s", out)
	}
	if !strings.Contains(out, "+$200") {
		t.Errorf("expected +$200 P/L for AAPL; got:\n%s", out)
	}
	if !strings.Contains(out, "—") {
		t.Errorf("expected em-dash for MSFT missing-price columns; got:\n%s", out)
	}
	if !strings.Contains(out, "Cost basis:") {
		t.Error("totals footer missing")
	}
}
