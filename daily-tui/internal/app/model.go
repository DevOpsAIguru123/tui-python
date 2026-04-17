package app

import (
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/calendar"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabWifi     = 0
	tabTodo     = 1
	tabCalendar = 2
)

var tabNames = []string{"WiFi", "Todo", "Calendar"}

// Model is the root Bubble Tea model.
type Model struct {
	activeTab int
	wifi      wifi.Model
	todo      todo.Model
	calendar  calendar.Model
	width     int
	height    int
	version   string
}

// New creates the root AppModel.
func New(wm wifi.Model, tm todo.Model, cm calendar.Model, version string) Model {
	return Model{wifi: wm, todo: tm, calendar: cm, version: version}
}

// ActiveTab returns the index of the currently active tab.
func (m Model) ActiveTab() int { return m.activeTab }

// Init starts all child models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.wifi.Init(), m.todo.Init(), m.calendar.Init())
}

// Update handles messages: tab-switch keys are consumed here; all others
// are forwarded to the active child.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// Only intercept navigation keys when the active child is not in an input mode.
		// If the user is typing a password or a todo, all keys belong to the child.
		childInputting := (m.activeTab == tabWifi && m.wifi.Inputting()) ||
			(m.activeTab == tabTodo && m.todo.Inputting()) ||
			(m.activeTab == tabCalendar && m.calendar.Inputting())
		if !childInputting {
			if msg.Type == tea.KeyRunes && string(msg.Runes) == "q" {
				return m, tea.Quit
			}
			if msg.Type == tea.KeyTab {
				m.activeTab = (m.activeTab + 1) % len(tabNames)
				return m, nil
			}
			if msg.Type == tea.KeyRunes {
				// When the Calendar tab is active, 1 / 2 / 3 switch its
				// time-range filters instead of tabs. The user can still
				// reach other tabs via Tab.
				if m.activeTab != tabCalendar {
					switch string(msg.Runes) {
					case "1":
						m.activeTab = tabWifi
						return m, nil
					case "2":
						m.activeTab = tabTodo
						return m, nil
					case "3":
						m.activeTab = tabCalendar
						return m, nil
					}
				}
			}
		}
	}

	// Async messages go to their owning tab regardless of which is active —
	// otherwise fetch results get lost if the user is on a different tab
	// when the command completes.
	if _, ok := msg.(calendar.EventsLoadedMsg); ok {
		updated, cmd := m.calendar.Update(msg)
		m.calendar = updated.(calendar.Model)
		return m, cmd
	}

	// Delegate remaining (mostly KeyMsg) messages to the active child.
	switch m.activeTab {
	case tabWifi:
		updated, cmd := m.wifi.Update(msg)
		m.wifi = updated.(wifi.Model)
		return m, cmd
	case tabTodo:
		updated, cmd := m.todo.Update(msg)
		m.todo = updated.(todo.Model)
		return m, cmd
	case tabCalendar:
		updated, cmd := m.calendar.Update(msg)
		m.calendar = updated.(calendar.Model)
		return m, cmd
	}
	return m, nil
}

// View renders the full TUI: tab bar + active panel.
func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(m.renderTabBar())
	sb.WriteString("\n\n")

	switch m.activeTab {
	case tabWifi:
		sb.WriteString(m.wifi.View())
	case tabTodo:
		sb.WriteString(m.todo.View())
	case tabCalendar:
		sb.WriteString(m.calendar.View())
	}

	sb.WriteString("\n\n")
	sb.WriteString(theme.HelpStyle.Render("tab/1/2/3 switch • q quit"))
	if m.version != "" {
		sb.WriteString("  " + theme.Dimmed.Render("v"+m.version))
	}
	return sb.String()
}

func (m Model) renderTabBar() string {
	var parts []string
	for i, name := range tabNames {
		if i == m.activeTab {
			parts = append(parts, theme.ActiveTab.Render(name))
		} else {
			parts = append(parts, theme.InactiveTab.Render(name))
		}
	}
	return strings.Join(parts, " ")
}
