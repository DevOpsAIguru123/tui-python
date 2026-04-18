package claudecode

import (
	"fmt"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// state captures what the tab is currently doing so the Update/View code can
// key off a single enum instead of sprinkling boolean flags everywhere.
type state int

const (
	stateBrowsing state = iota
	stateRunning
	stateDone
)

// Model is the Bubble Tea model for the Claude Code tab.
type Model struct {
	commands []Command
	cursor   int
	state    state
	selected Command
	output   string
	err      string
	startAt  time.Time
	finishAt time.Time
	rowWidth int
}

// New builds a fresh Model and loads the initial command list from disk.
// Discovery is cheap (a small WalkDir on ~/.claude/commands) so it's fine
// to run synchronously in the constructor.
func New() Model {
	return Model{commands: Discover()}
}

// Inputting reports whether the tab owns all keys. It does during an active
// run so the user can't tab-switch out of a pending job; otherwise the app
// chrome can intercept tab/number keys as usual.
func (m Model) Inputting() bool { return m.state == stateRunning }

// Commands exposes the loaded command list for tests.
func (m Model) Commands() []Command { return m.commands }

// Cursor exposes the current highlight index for tests.
func (m Model) Cursor() int { return m.cursor }

// State exposes the current run state for tests.
func (m Model) State() state { return m.state }

// Output returns the last completed run's combined stdout/stderr, or "" if
// no run has completed yet.
func (m Model) Output() string { return m.output }

// Count reports the number of discovered commands (sidebar badge, if used).
func (m Model) Count() int { return len(m.commands) }

// Title is the subtitle shown in the breadcrumb header.
func (m Model) Title() string {
	switch m.state {
	case stateRunning:
		return "running · " + m.selected.Name
	case stateDone:
		if m.err != "" {
			return "error · " + m.selected.Name
		}
		return "done · " + m.selected.Name
	default:
		return fmt.Sprintf("%d command%s", len(m.commands), plural(len(m.commands)))
	}
}

// Help returns the key hints shown in the app's bottom help bar.
func (m Model) Help() []theme.KeyHint {
	switch m.state {
	case stateRunning:
		return []theme.KeyHint{{Key: "…", Label: "running"}}
	case stateDone:
		return []theme.KeyHint{
			{Key: "enter", Label: "run again"},
			{Key: "esc", Label: "back"},
		}
	default:
		return []theme.KeyHint{
			{Key: "↑↓", Label: "navigate"},
			{Key: "enter", Label: "run"},
		}
	}
}

// SetRowWidth accepts the main-pane content width from the app; the View
// uses it to soft-wrap long output so it doesn't overflow the pane.
func (m *Model) SetRowWidth(w int) { m.rowWidth = w }

// Init is a no-op — Discover already ran inside New().
func (m Model) Init() tea.Cmd { return nil }

// Update drives the tab's state machine.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RunDoneMsg:
		m.state = stateDone
		m.output = msg.Output
		m.finishAt = time.Now()
		if msg.Err != nil {
			m.err = msg.Err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While a run is in-flight the tab swallows all keys so the user can't
	// accidentally launch a second run or navigate away mid-stream.
	if m.state == stateRunning {
		return m, nil
	}
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.commands)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if len(m.commands) == 0 {
			return m, nil
		}
		m.selected = m.commands[m.cursor]
		m.state = stateRunning
		m.output = ""
		m.err = ""
		m.startAt = time.Now()
		return m, RunCommandCmd(m.selected.Prompt)
	case tea.KeyEsc:
		if m.state == stateDone {
			m.state = stateBrowsing
			m.output = ""
			m.err = ""
		}
	}
	return m, nil
}

// View renders the tab's content (no help bar — the app chrome draws that).
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(m.renderHeader())
	sb.WriteString("\n\n")

	sb.WriteString(theme.Dimmed.Render("── COMMANDS ") +
		theme.Dimmed.Render(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	if len(m.commands) == 0 {
		sb.WriteString(theme.Dimmed.Render("  no commands found") + "\n")
	} else {
		for i, c := range m.commands {
			sb.WriteString(m.renderRow(c, i == m.cursor) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.renderOutput())
	return sb.String()
}

// renderHeader reports either the idle hint or the running/done status for
// the in-flight command, with elapsed wall-clock time.
func (m Model) renderHeader() string {
	switch m.state {
	case stateRunning:
		elapsed := time.Since(m.startAt).Truncate(time.Second)
		return theme.Dimmed.Render("● ") +
			theme.Normal.Render("running ") +
			theme.ConnectedText.Render(m.selected.Prompt) +
			"  " + theme.Dimmed.Render("("+elapsed.String()+" elapsed)")
	case stateDone:
		dur := m.finishAt.Sub(m.startAt).Truncate(time.Millisecond)
		if m.err != "" {
			return theme.StatusErr.Render("✗ ") +
				theme.Normal.Render(m.selected.Prompt) +
				"  " + theme.Dimmed.Render("("+dur.String()+" · failed)")
		}
		return theme.ConnectedDot.Render("● ") +
			theme.Normal.Render(m.selected.Prompt) +
			"  " + theme.Dimmed.Render("("+dur.String()+")")
	default:
		return theme.Dimmed.Render("pick a command and press ") +
			theme.ConnectedText.Render("enter") +
			theme.Dimmed.Render(" to run via ") +
			theme.ConnectedText.Render("claude -p")
	}
}

func (m Model) renderRow(c Command, selected bool) string {
	caret := "  "
	if selected {
		caret = theme.CrumbCaret.Render("▸ ")
	}
	name := theme.Normal.Render(c.Name)
	if selected {
		name = theme.RowSelected.Render(c.Name)
	}
	tag := theme.Tag.Render(c.Source)
	prompt := theme.Dimmed.Render(c.Prompt)
	return caret + name + "  " + tag + "  " + prompt
}

// renderOutput draws whatever's appropriate below the command list:
// a spinner-less "running" notice, the error text, or the captured output
// soft-wrapped to the pane width so long responses don't overflow.
func (m Model) renderOutput() string {
	switch m.state {
	case stateRunning:
		return theme.Dimmed.Render("  waiting for claude…")
	case stateDone:
		var sb strings.Builder
		sb.WriteString(theme.Dimmed.Render("── OUTPUT ") +
			theme.Dimmed.Render(strings.Repeat("─", 62)))
		sb.WriteString("\n")
		if m.err != "" {
			sb.WriteString(theme.StatusErr.Render("  " + m.err))
			sb.WriteString("\n")
		}
		if m.output != "" {
			sb.WriteString(wrapOutput(m.output, m.rowWidth))
			sb.WriteString("\n")
		}
		return sb.String()
	}
	return ""
}

func wrapOutput(text string, width int) string {
	if width <= 0 {
		return text
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
