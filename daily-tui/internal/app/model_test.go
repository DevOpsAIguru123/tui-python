package app_test

import (
	"path/filepath"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/app"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/calendar"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/claudecode"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/portfolio"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

func newApp(t *testing.T) app.Model {
	t.Helper()
	cfg := &config.Config{}
	wm := wifi.New(cfg)
	tm := todo.New(todo.NewStore(filepath.Join(t.TempDir(), "todos.json")))
	cm := calendar.New()
	pm := portfolio.New(portfolio.NewStore(filepath.Join(t.TempDir(), "portfolio.json")))
	cc := claudecode.New(config.ClaudeCodeConfig{})
	return app.New(wm, tm, cm, pm, cc, "test")
}

func TestAppModelDefaultTab(t *testing.T) {
	m := newApp(t)
	if m.ActiveTab() != 0 {
		t.Errorf("expected default tab 0 (WiFi), got %d", m.ActiveTab())
	}
}

func TestAppModelTabSwitch(t *testing.T) {
	m := newApp(t)
	// With five tabs, Tab cycles WiFi→Todo→Calendar→Portfolio→Claude→WiFi.
	for i, want := range []int{1, 2, 3, 4, 0} {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(app.Model)
		if m.ActiveTab() != want {
			t.Errorf("after Tab #%d: got %d, want %d", i+1, m.ActiveTab(), want)
		}
	}
}

func TestClaudeTabRemainsNavigableAfterRun(t *testing.T) {
	// Regression: a user reported "i could not navigate after command ran".
	// After a RunDoneMsg lands on the Claude tab, the global Tab key and
	// number keys must still cycle to other tabs. Inputting() always
	// returning false on Claude is what guarantees this.
	m := newApp(t)
	// Switch to Claude tab.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = updated.(app.Model)
	if m.ActiveTab() != 4 {
		t.Fatalf("setup: expected to be on Claude tab (4), got %d", m.ActiveTab())
	}
	// Press Enter to start a run, then deliver a RunDoneMsg manually.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(app.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Prompt: "hello", Output: "hi"})
	m = updated.(app.Model)
	// Tab from done-state Claude must wrap to WiFi (tab 0).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(app.Model)
	if m.ActiveTab() != 0 {
		t.Errorf("expected Tab from done-Claude to go to WiFi (0), got %d", m.ActiveTab())
	}
	// Number key should also work — '3' from WiFi switches to Calendar.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(app.Model)
	if m.ActiveTab() != 2 {
		t.Errorf("expected '3' from WiFi to go to Calendar (2), got %d", m.ActiveTab())
	}
}

func TestAppModelNumberKeySwitch(t *testing.T) {
	m := newApp(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	am := updated.(app.Model)
	if am.ActiveTab() != 1 {
		t.Errorf("expected tab 1 after '2', got %d", am.ActiveTab())
	}
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 2 {
		t.Errorf("expected tab 2 after '3', got %d", am.ActiveTab())
	}
	// On the Calendar tab, '1' / '2' / '3' are claimed by the child for
	// view filters — they must NOT switch tabs. Tab key still does.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 2 {
		t.Errorf("expected calendar tab to consume '1'; tab changed to %d", am.ActiveTab())
	}
	// Tab from Calendar goes to Portfolio (tab 3), not WiFi.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model)
	if am.ActiveTab() != 3 {
		t.Errorf("expected Portfolio tab (3) after Tab from Calendar, got %d", am.ActiveTab())
	}
	// Portfolio also claims '1' / '2' for section switching.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 3 {
		t.Errorf("expected Portfolio tab to consume '1'; tab changed to %d", am.ActiveTab())
	}
	// Tab from Portfolio advances to Claude (tab 4).
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model)
	if am.ActiveTab() != 4 {
		t.Errorf("expected Claude tab (4) after Tab from Portfolio, got %d", am.ActiveTab())
	}
	// Tab from Claude wraps back to WiFi.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model)
	if am.ActiveTab() != 0 {
		t.Errorf("expected wrap to tab 0 after Tab from Claude, got %d", am.ActiveTab())
	}
	// From WiFi, '4' switches to Portfolio and '5' to Claude.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 3 {
		t.Errorf("expected tab 3 after '4' from WiFi, got %d", am.ActiveTab())
	}
	// '5' from Portfolio is claimed by the tab; need to go back to WiFi first.
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model)
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyTab})
	am = updated.(app.Model) // now at WiFi
	updated, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	am = updated.(app.Model)
	if am.ActiveTab() != 4 {
		t.Errorf("expected tab 4 after '5' from WiFi, got %d", am.ActiveTab())
	}
}
