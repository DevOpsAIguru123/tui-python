package app

import (
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabWifi = 0
	tabTodo = 1
)

var tabNames = []string{"WiFi", "Todo"}

// Model is the root Bubble Tea model.
type Model struct {
	activeTab int
	wifi      wifi.Model
	todo      todo.Model
	width     int
	height    int
}

// New creates the root AppModel.
func New(wm wifi.Model, tm todo.Model) Model {
	return Model{wifi: wm, todo: tm}
}

// ActiveTab returns the index of the currently active tab.
func (m Model) ActiveTab() int { return m.activeTab }

// Init starts both child models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.wifi.Init(), m.todo.Init())
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
			(m.activeTab == tabTodo && m.todo.Adding())
		if !childInputting {
			if msg.Type == tea.KeyRunes && string(msg.Runes) == "q" {
				return m, tea.Quit
			}
			if msg.Type == tea.KeyTab {
				m.activeTab = (m.activeTab + 1) % len(tabNames)
				return m, nil
			}
			if msg.Type == tea.KeyRunes {
				switch string(msg.Runes) {
				case "1":
					m.activeTab = tabWifi
					return m, nil
				case "2":
					m.activeTab = tabTodo
					return m, nil
				}
			}
		}
	}

	// Delegate to active child
	switch m.activeTab {
	case tabWifi:
		updated, cmd := m.wifi.Update(msg)
		m.wifi = updated.(wifi.Model)
		return m, cmd
	case tabTodo:
		updated, cmd := m.todo.Update(msg)
		m.todo = updated.(todo.Model)
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
	}

	sb.WriteString("\n\n")
	sb.WriteString(theme.HelpStyle.Render("tab/1/2 switch • q quit"))
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
