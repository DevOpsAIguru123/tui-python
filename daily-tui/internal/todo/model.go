package todo

import (
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the Todo tab.
type Model struct {
	store  *Store
	tasks  []Task
	cursor int
	adding bool
	input  textinput.Model
}

// New creates a TodoModel backed by the given store.
func New(s *Store) Model {
	ti := textinput.New()
	ti.Placeholder = "Add a task..."
	ti.CharLimit = 200
	return Model{store: s, tasks: s.All(), input: ti}
}

// Exported accessors for tests
func (m Model) Adding() bool  { return m.adding }
func (m Model) Tasks() []Task { return m.tasks }
func (m Model) Cursor() int   { return m.cursor }

// Init is a no-op.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.adding {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.adding {
		switch msg.Type {
		case tea.KeyEsc:
			m.adding = false
			m.input.Blur()
			m.input.SetValue("")
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				m.store.Add(text)
				m.tasks = m.store.All()
				m.cursor = len(m.tasks) - 1
			}
			m.adding = false
			m.input.Blur()
			m.input.SetValue("")
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case tea.KeySpace:
		if len(m.tasks) > 0 {
			m.store.Toggle(m.tasks[m.cursor].ID)
			m.tasks = m.store.All()
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "a":
			m.adding = true
			m.input.Focus()
		case "d":
			if len(m.tasks) > 0 {
				m.store.Delete(m.tasks[m.cursor].ID)
				m.tasks = m.store.All()
				if m.cursor >= len(m.tasks) && m.cursor > 0 {
					m.cursor--
				}
			}
		}
	}
	return m, nil
}

// View renders the Todo tab content.
func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(theme.Dimmed.Render("Today's Tasks") + "\n\n")

	if len(m.tasks) == 0 {
		sb.WriteString(theme.Dimmed.Render("  No tasks yet — press 'a' to add one") + "\n")
	} else {
		for i, t := range m.tasks {
			checkbox := "☐"
			style := theme.Normal
			if t.Done {
				checkbox = "☑"
				style = theme.Dimmed
			}
			line := checkbox + "  " + t.Text
			if i == m.cursor && !m.adding {
				sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
			} else {
				sb.WriteString(style.Render("  "+line) + "\n")
			}
		}
	}

	if m.adding {
		sb.WriteString("\n" + m.input.View() + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("↑↓ navigate • space complete • d delete • a add"))
	return sb.String()
}
