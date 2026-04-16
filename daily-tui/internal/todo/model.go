package todo

import (
	"fmt"
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the Todo tab.
type Model struct {
	store      *Store
	tasks      []Task
	cursor     int
	adding     bool
	editing    bool
	editTaskID string
	input      textinput.Model
}

// New creates a TodoModel backed by the given store.
func New(s *Store) Model {
	ti := textinput.New()
	ti.Placeholder = "Add a task..."
	ti.CharLimit = 200
	return Model{store: s, tasks: s.All(), input: ti}
}

// Exported accessors for tests
func (m Model) Adding() bool    { return m.adding }
func (m Model) Editing() bool   { return m.editing }
func (m Model) Inputting() bool { return m.adding || m.editing }
func (m Model) Tasks() []Task   { return m.tasks }
func (m Model) Cursor() int     { return m.cursor }

// Init is a no-op.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.adding || m.editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.adding || m.editing {
		switch msg.Type {
		case tea.KeyEsc:
			m.adding = false
			m.editing = false
			m.editTaskID = ""
			m.input.Blur()
			m.input.SetValue("")
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				if m.adding {
					m.store.Add(text)
					m.tasks = m.store.All()
					m.cursor = len(m.tasks) - 1
				} else if m.editing {
					m.store.Update(m.editTaskID, text)
					m.tasks = m.store.All()
				}
			}
			m.adding = false
			m.editing = false
			m.editTaskID = ""
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
			pending, completed := splitTasks(m.tasks)
			displayed := append(pending, completed...)
			m.store.Toggle(displayed[m.cursor].ID)
			m.tasks = m.store.All()
			if m.cursor >= len(m.tasks) {
				m.cursor = len(m.tasks) - 1
			}
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "a":
			m.adding = true
			m.input.Placeholder = "Add a task..."
			m.input.Focus()
		case "e":
			if len(m.tasks) > 0 {
				pending, completed := splitTasks(m.tasks)
				displayed := append(pending, completed...)
				m.editing = true
				m.editTaskID = displayed[m.cursor].ID
				m.input.Placeholder = "Edit task..."
				m.input.SetValue(displayed[m.cursor].Text)
				m.input.Focus()
			}
		case "d":
			if len(m.tasks) > 0 {
				pending, completed := splitTasks(m.tasks)
				displayed := append(pending, completed...)
				m.store.Delete(displayed[m.cursor].ID)
				m.tasks = m.store.All()
				if m.cursor >= len(m.tasks) && m.cursor > 0 {
					m.cursor--
				}
			}
		}
	}
	return m, nil
}

// splitTasks divides tasks into pending and completed, preserving insertion order.
func splitTasks(tasks []Task) (pending, completed []Task) {
	for _, t := range tasks {
		if t.Done {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}
	return
}

// renderProgressBar returns a 30-char progress bar with percentage.
func renderProgressBar(done, total int) string {
	if total == 0 {
		return ""
	}
	const width = 30
	filled := width * done / total
	bar := theme.ProgressFilled.Render(strings.Repeat("▓", filled)) +
		theme.ProgressEmpty.Render(strings.Repeat("░", width-filled))
	pct := fmt.Sprintf("%d%%", 100*done/total)
	return bar + "  " + theme.Dimmed.Render(pct)
}

// View renders the Todo tab content with sectioned layout.
func (m Model) View() string {
	var sb strings.Builder

	pending, completed := splitTasks(m.tasks)
	doneCount := len(completed)
	total := len(m.tasks)

	// Header with task count
	header := theme.Dimmed.Render("Today's Tasks")
	if total > 0 {
		header += "  " + theme.Badge.Render(fmt.Sprintf("%d/%d done", doneCount, total))
	}
	sb.WriteString(header + "\n")

	// Progress bar
	if total > 0 {
		sb.WriteString(renderProgressBar(doneCount, total) + "\n")
	}
	sb.WriteString("\n")

	if total == 0 {
		sb.WriteString(theme.Dimmed.Render("  No tasks yet — press 'a' to add one") + "\n")
	} else {
		// Pending section
		sb.WriteString(theme.SectionHeader.Render(fmt.Sprintf("── Pending (%d) ", len(pending))) +
			theme.Dimmed.Render("──────────────────────") + "\n")
		if len(pending) == 0 {
			sb.WriteString(theme.Dimmed.Render("  No pending tasks") + "\n")
		} else {
			for i, t := range pending {
				line := "☐  " + t.Text
				if i == m.cursor && !m.adding && !m.editing {
					sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
				} else {
					sb.WriteString(theme.Normal.Render("  "+line) + "\n")
				}
			}
		}
		sb.WriteString("\n")

		// Completed section
		sb.WriteString(theme.Dimmed.Render(fmt.Sprintf("── Completed (%d) ──────────────────", len(completed))) + "\n")
		if len(completed) == 0 {
			sb.WriteString(theme.Dimmed.Render("  No completed tasks yet") + "\n")
		} else {
			for i, t := range completed {
				line := "☑  " + t.Text
				cursorIdx := len(pending) + i
				if cursorIdx == m.cursor && !m.adding && !m.editing {
					sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
				} else {
					sb.WriteString(theme.Dimmed.Render("  "+line) + "\n")
				}
			}
		}
	}

	// Input area
	if m.adding || m.editing {
		sb.WriteString("\n" + m.input.View() + "\n")
	}

	// Help bar
	sb.WriteString("\n")
	if m.adding {
		sb.WriteString(theme.HelpStyle.Render("enter confirm • esc cancel"))
	} else if m.editing {
		sb.WriteString(theme.HelpStyle.Render("enter save • esc cancel"))
	} else {
		sb.WriteString(theme.HelpStyle.Render("↑↓ nav • space done • e edit • d del • a add"))
	}
	return sb.String()
}
