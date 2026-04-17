package calendar

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func fakeEvents() []Event {
	day := time.Date(2026, 4, 17, 9, 0, 0, 0, time.Local)
	return []Event{
		{Calendar: "Work", Title: "Standup", Start: day, End: day.Add(15 * time.Minute)},
		{Calendar: "Personal", Title: "Dentist", Start: day.Add(5 * time.Hour), End: day.Add(6 * time.Hour), Location: "Downtown"},
	}
}

func TestView_loadingInitial(t *testing.T) {
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
	m := New()
	next, _ := m.Update(EventsLoadedMsg{Events: fakeEvents()})
	m = next.(Model)

	out := m.View()
	for _, want := range []string{"Standup", "Dentist", "Downtown", "Work", "Personal"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q; got:\n%s", want, out)
		}
	}
}

func TestView_empty(t *testing.T) {
	m := New()
	next, _ := m.Update(EventsLoadedMsg{Events: nil})
	m = next.(Model)
	if !strings.Contains(m.View(), "No events") {
		t.Errorf("expected empty-state message; got: %s", m.View())
	}
}

func TestView_error(t *testing.T) {
	m := New()
	next, _ := m.Update(EventsLoadedMsg{Err: errors.New("boom")})
	m = next.(Model)
	out := m.View()
	if !strings.Contains(out, "Failed to load") || !strings.Contains(out, "boom") {
		t.Errorf("expected error state with 'boom'; got: %s", out)
	}
}

func TestUpdate_cursorNavigation(t *testing.T) {
	m := New()
	next, _ := m.Update(EventsLoadedMsg{Events: fakeEvents()})
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
	// Calendar is read-only; no input modes exist.
	if New().Inputting() {
		t.Error("Calendar.Inputting() should always be false")
	}
}
