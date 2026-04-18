package claudecode

import (
	"fmt"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/viewport"
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

	contentWidth  int
	contentHeight int
	vp            viewport.Model
}

// New builds a fresh Model and loads the initial command list from disk.
// Discovery is cheap (a small WalkDir on ~/.claude/commands) so it's fine
// to run synchronously in the constructor.
func New() Model {
	return Model{commands: Discover(), vp: viewport.New(0, 0)}
}

// Inputting reports whether the tab owns all keys. The Claude tab never
// takes raw text input (runs are fire-and-forget), so this is always false
// — the user is free to tab away to another pane while a command is still
// executing. The RunDoneMsg is routed back to this tab regardless of where
// the user is when the run finishes, so nothing is lost by switching away.
//
// While a run is in-flight the tab's own handleKey still swallows keystrokes
// that arrive on the active pane (Enter, arrow keys) so the user can't
// accidentally launch a second run; returning false here just means the
// app chrome's tab/number keys keep working.
func (m Model) Inputting() bool { return false }

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

// Help returns the key hints shown in the app's bottom help bar. During the
// "done" state the hints acknowledge that ↑↓/PgUp/PgDn scroll the output
// viewport rather than the command list.
func (m Model) Help() []theme.KeyHint {
	switch m.state {
	case stateRunning:
		return []theme.KeyHint{{Key: "…", Label: "running"}}
	case stateDone:
		return []theme.KeyHint{
			{Key: "↑↓", Label: "scroll"},
			{Key: "pgup/dn", Label: "page"},
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
func (m *Model) SetRowWidth(w int) {
	m.contentWidth = w
	m.resizeViewport()
}

// SetContentHeight accepts the main-pane usable height from the app. The
// output viewport sizes itself so the rest of the UI (breadcrumb, command
// list, help bar) stays visible no matter how long the captured output is.
func (m *Model) SetContentHeight(h int) {
	m.contentHeight = h
	m.resizeViewport()
}

// resizeViewport recomputes the viewport dimensions from the latest width
// and height. The viewport's height is the content height minus room for
// the fixed chrome we render above it (header, command list, OUTPUT
// divider) — anything left over after that goes to the scrollable output.
func (m *Model) resizeViewport() {
	if m.contentWidth <= 0 {
		return
	}
	// Reserve vertical room for the pieces we draw above the viewport:
	//   1 line breadcrumb header (drawn by app) — NOT counted here
	//   1 line tab header
	//   1 blank line
	//   1 line "── COMMANDS ──" divider
	//   N lines for the command rows
	//   1 blank line
	//   1 line "── OUTPUT ──" divider
	// plus a little slack so the help bar at the bottom still fits.
	chrome := 7 + len(m.commands)
	h := m.contentHeight - chrome
	if h < 3 {
		h = 3
	}
	m.vp.Width = m.contentWidth
	m.vp.Height = h
}

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
		m.resizeViewport()
		m.vp.SetContent(m.output)
		m.vp.GotoTop()
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

	// In the "done" state, arrow + page keys drive the output viewport
	// instead of the command-list cursor — the user is reading output, not
	// re-picking a command. Enter re-runs the last command, Esc goes back.
	if m.state == stateDone {
		switch msg.Type {
		case tea.KeyEnter:
			return m.startRun()
		case tea.KeyEsc:
			m.state = stateBrowsing
			m.output = ""
			m.err = ""
			m.vp.SetContent("")
			return m, nil
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
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
		return m.startRun()
	}
	return m, nil
}

// startRun transitions to the running state and fires the subprocess.
// Extracted so both the browsing-state Enter and the done-state "run again"
// Enter go through the same code path.
func (m Model) startRun() (tea.Model, tea.Cmd) {
	if len(m.commands) == 0 {
		return m, nil
	}
	m.selected = m.commands[m.cursor]
	m.state = stateRunning
	m.output = ""
	m.err = ""
	m.startAt = time.Now()
	m.vp.SetContent("")
	return m, RunCommandCmd(m.selected.Prompt, m.selected.ExtraArgs...)
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
	row := caret + name + "  " + tag + "  " + prompt

	// Make it obvious up-front when an entry carries "dangerous" flags like
	// --dangerously-skip-permissions. Flag-free commands render unchanged.
	if len(c.ExtraArgs) > 0 {
		flags := strings.Join(c.ExtraArgs, " ")
		if hasDangerousFlag(c.ExtraArgs) {
			row += "  " + theme.StatusErr.Render(flags)
		} else {
			row += "  " + theme.Dimmed.Render(flags)
		}
	}
	return row
}

func hasDangerousFlag(args []string) bool {
	for _, a := range args {
		if strings.Contains(a, "dangerously") {
			return true
		}
	}
	return false
}

// renderOutput draws whatever's appropriate below the command list. In the
// "done" state the output is rendered through a viewport so long Claude
// responses scroll locally instead of pushing the sidebar and command list
// off the top of the screen.
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
			sb.WriteString(m.vp.View())
			sb.WriteString("\n")
			sb.WriteString(m.scrollHint())
		}
		return sb.String()
	}
	return ""
}

// scrollHint summarises whether there's more output above or below the
// current viewport position. Shown below the viewport in dim text.
func (m Model) scrollHint() string {
	if m.vp.TotalLineCount() <= m.vp.Height {
		return ""
	}
	pct := m.vp.ScrollPercent() * 100
	return theme.Dimmed.Render(fmt.Sprintf(
		"  %d–%d / %d lines · %.0f%%",
		m.vp.YOffset+1,
		m.vp.YOffset+m.vp.Height,
		m.vp.TotalLineCount(),
		pct,
	))
}

// wrapOutput is retained for tests that may rely on soft-wrapping behavior.
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
