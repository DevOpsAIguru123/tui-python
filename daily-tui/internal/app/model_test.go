package app_test

import (
	"path/filepath"
	"testing"

	"daily-tui/internal/app"
	"daily-tui/internal/config"
	"daily-tui/internal/todo"
	"daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

func newApp(t *testing.T) app.Model {
	t.Helper()
	cfg := &config.Config{}
	wm := wifi.New(cfg)
	tm := todo.New(todo.NewStore(filepath.Join(t.TempDir(), "todos.json")))
	return app.New(wm, tm)
}

func TestAppModelDefaultTab(t *testing.T) {
	m := newApp(t)
	if m.ActiveTab() != 0 {
		t.Errorf("expected default tab 0 (WiFi), got %d", m.ActiveTab())
	}
}

func TestAppModelTabSwitch(t *testing.T) {
	m := newApp(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	am := updated.(app.Model)
	if am.ActiveTab() != 1 {
		t.Errorf("expected tab 1 after Tab, got %d", am.ActiveTab())
	}
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model)
	if am.ActiveTab() != 0 {
		t.Errorf("expected tab 0 after second Tab, got %d", am.ActiveTab())
	}
}

func TestAppModelNumberKeySwitch(t *testing.T) {
	m := newApp(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	am := updated.(app.Model)
	if am.ActiveTab() != 1 {
		t.Errorf("expected tab 1 after '2', got %d", am.ActiveTab())
	}
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 0 {
		t.Errorf("expected tab 0 after '1', got %d", am.ActiveTab())
	}
}
