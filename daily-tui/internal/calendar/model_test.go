package calendar

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// withIsolatedCache points cacheBaseDir at a fresh temp dir for the lifetime
// of a test so persisted cache files from other tests or the host machine
// can't leak into the Model under test.
func withIsolatedCache(t *testing.T) {
	t.Helper()
	prev := cacheBaseDir
	cacheBaseDir = t.TempDir()
	t.Cleanup(func() { cacheBaseDir = prev })
}

func fakeEvents() []Event {
	day := time.Date(2026, 4, 17, 9, 0, 0, 0, time.Local)
	return []Event{
		{Calendar: "Work", Title: "Standup", Start: day, End: day.Add(15 * time.Minute)},
		{Calendar: "Personal", Title: "Dentist", Start: day.Add(5 * time.Hour), End: day.Add(6 * time.Hour), Location: "Downtown"},
	}
}

func TestView_loadingInitial(t *testing.T) {
	withIsolatedCache(t)
	m := New()
	out := m.View()
	// On darwin the model is loading; on other platforms the View short-circuits
	// to the unsupported-platform message. Either outcome is acceptable here.
	if Supported() {
		if !strings.Contains(out, "Loading") {
			t.Errorf("expected loading state, got: %s", out)
		}
	} else {
		if !strings.Contains(out, "macOS-only") {
			t.Errorf("expected unsupported-platform message, got: %s", out)
		}
	}
}

func TestView_withEvents(t *testing.T) {
	withIsolatedCache(t)
	m := New()
	next, _ := m.Update(EventsLoadedMsg{View: m.view, Events: fakeEvents(), FetchedAt: time.Now()})
	m = next.(Model)

	out := m.View()
	for _, want := range []string{"Standup", "Dentist", "Downtown", "Work", "Personal"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q; got:\n%s", want, out)
		}
	}
}

func TestView_empty(t *testing.T) {
	if !Supported() {
		t.Skip("empty-state text only renders when platform supports calendar")
	}
	withIsolatedCache(t)
	m := New()
	next, _ := m.Update(EventsLoadedMsg{View: m.view, Events: nil})
	m = next.(Model)
	if !strings.Contains(m.View(), "No events") {
		t.Errorf("expected empty-state message; got: %s", m.View())
	}
}

func TestView_error(t *testing.T) {
	if !Supported() {
		t.Skip("error text only renders when platform supports calendar")
	}
	withIsolatedCache(t)
	m := New()
	next, _ := m.Update(EventsLoadedMsg{View: m.view, Err: errors.New("boom")})
	m = next.(Model)
	out := m.View()
	if !strings.Contains(out, "Failed to load") || !strings.Contains(out, "boom") {
		t.Errorf("expected error state with 'boom'; got: %s", out)
	}
}

func TestUpdate_cursorNavigation(t *testing.T) {
	withIsolatedCache(t)
	m := New()
	next, _ := m.Update(EventsLoadedMsg{View: m.view, Events: fakeEvents()})
	m = next.(Model)

	if m.cursor != 0 {
		t.Fatalf("initial cursor should be 0, got %d", m.cursor)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor after Down should be 1, got %d", m.cursor)
	}
	// Down past end should stay clamped.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor should clamp at 1, got %d", m.cursor)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor after Up should be 0, got %d", m.cursor)
	}
}

func TestInputting_alwaysFalse(t *testing.T) {
	withIsolatedCache(t)
	// Calendar is read-only; no input modes exist.
	if New().Inputting() {
		t.Error("Calendar.Inputting() should always be false")
	}
}

func TestNew_defaultsToTodayView(t *testing.T) {
	withIsolatedCache(t)
	if got := New().CurrentView(); got != ViewToday {
		t.Errorf("default view = %v, want ViewToday", got)
	}
}

func TestUpdate_switchView(t *testing.T) {
	withIsolatedCache(t)
	cases := []struct {
		key  rune
		want View
	}{
		{'1', ViewToday},
		{'2', ViewWeek},
		{'3', ViewMonth},
	}
	for _, tc := range cases {
		m := New()
		// Move off default so "1" is an actual change and exercises the path.
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = next.(Model)
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		m = next.(Model)
		if m.CurrentView() != tc.want {
			t.Errorf("key %q → view %v, want %v", tc.key, m.CurrentView(), tc.want)
		}
	}
}

func TestUpdate_switchView_hydratesFromCache(t *testing.T) {
	withIsolatedCache(t)
	// Seed a fresh cache for the "week" view.
	cached := fakeEvents()
	SaveCache(ViewWeek, cached)

	m := New()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = next.(Model)

	if m.CurrentView() != ViewWeek {
		t.Fatalf("view = %v, want ViewWeek", m.CurrentView())
	}
	if len(m.Events()) != len(cached) {
		t.Errorf("events not hydrated from cache: got %d, want %d", len(m.Events()), len(cached))
	}
	if m.loading {
		t.Error("switching to a view with a fresh cache should not enter loading state")
	}
}

func TestUpdate_ignoresStaleViewMessage(t *testing.T) {
	withIsolatedCache(t)
	m := New()
	// User moved on to the month view.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = next.(Model)

	// A late response for the original (today) view arrives.
	next, _ = m.Update(EventsLoadedMsg{View: ViewToday, Events: fakeEvents()})
	m = next.(Model)

	if len(m.Events()) != 0 {
		t.Errorf("stale view response leaked into current view: got %d events", len(m.Events()))
	}
}

func TestNew_hydratesFromFreshCache(t *testing.T) {
	if !Supported() {
		t.Skip("loading flag on non-darwin is always false; hydration path not exercised")
	}
	withIsolatedCache(t)
	SaveCache(ViewToday, fakeEvents())
	m := New()
	if m.loading {
		t.Error("fresh cache hit should skip the loading flag")
	}
	if len(m.Events()) == 0 {
		t.Error("expected events hydrated from cache")
	}
}
