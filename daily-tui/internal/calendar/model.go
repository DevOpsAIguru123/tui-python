package calendar

import (
	"fmt"
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultDaysAhead = 7

// Model is the Bubble Tea model for the Calendar tab.
type Model struct {
	events  []Event
	cursor  int
	loading bool
	err     error
}

// New creates a Calendar model. Fetch is kicked off by Init.
func New() Model {
	return Model{loading: true}
}

// Inputting reports whether the tab owns all keys (none for read-only view).
func (m Model) Inputting() bool { return false }

// Events returns the currently loaded events (used by tests).
func (m Model) Events() []Event { return m.events }

// Init kicks off the initial fetch (or returns nil on unsupported OS).
func (m Model) Init() tea.Cmd {
	if !Supported() {
		return nil
	}
	return FetchEventsCmd(defaultDaysAhead)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case EventsLoadedMsg:
		m.loading = false
		m.err = msg.Err
		m.events = msg.Events
		if m.cursor >= len(m.events) {
			m.cursor = 0
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.events)-1 {
				m.cursor++
			}
		case tea.KeyRunes:
			if string(msg.Runes) == "r" && Supported() {
				m.loading = true
				m.err = nil
				return m, FetchEventsCmd(defaultDaysAhead)
			}
		}
	}
	return m, nil
}

// View renders the Calendar tab.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(theme.Dimmed.Render("Upcoming — next 7 days"))
	sb.WriteString("\n\n")

	if m.loading {
		if !Supported() {
			sb.WriteString(theme.StatusErr.Render("Calendar tab is macOS-only (uses Calendar.app via osascript)."))
			sb.WriteString("\n")
			return sb.String()
		}
		sb.WriteString(theme.Dimmed.Render("  Loading events..."))
		sb.WriteString("\n\n")
		sb.WriteString(theme.HelpStyle.Render("r refresh"))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(theme.StatusErr.Render("Failed to load: " + m.err.Error()))
		sb.WriteString("\n")
		sb.WriteString(theme.Dimmed.Render("  First run may need to grant Terminal access to Calendar in System Settings → Privacy."))
		sb.WriteString("\n\n")
		sb.WriteString(theme.HelpStyle.Render("r retry"))
		return sb.String()
	}

	if len(m.events) == 0 {
		sb.WriteString(theme.Dimmed.Render("  No events in the next 7 days"))
		sb.WriteString("\n\n")
		sb.WriteString(theme.HelpStyle.Render("r refresh"))
		return sb.String()
	}

	var lastDay string
	for i, e := range m.events {
		day := e.Start.Format("Mon, Jan 2")
		if day != lastDay {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(theme.SectionHeader.Render("── "+day+" ") +
				theme.Dimmed.Render("──────────────────────") + "\n")
			lastDay = day
		}
		sb.WriteString(renderEventLine(e, i == m.cursor))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("↑↓ nav • r refresh"))
	return sb.String()
}

func renderEventLine(e Event, selected bool) string {
	var when string
	if e.IsAllDay() {
		when = "all-day"
	} else {
		when = fmt.Sprintf("%s–%s",
			e.Start.Format("3:04pm"),
			e.End.Format("3:04pm"))
	}
	line := fmt.Sprintf("%-13s %s", when, e.Title)
	if e.Location != "" {
		line += theme.Dimmed.Render("  @ " + e.Location)
	}
	if e.Calendar != "" {
		line += "  " + theme.Badge.Render(e.Calendar)
	}
	if selected {
		return theme.Selected.Render("▸ " + line)
	}
	return theme.Normal.Render("  " + line)
}
