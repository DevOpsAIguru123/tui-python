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

// runStatus is the per-command lifecycle. Each row in the launcher list
// owns its own runStatus so several commands can be in flight at once.
type runStatus int

const (
	runIdle runStatus = iota
	runRunning
	runDone
)

// runState tracks a single command's most recent invocation. We keep one
// per command name (in Model.runs) so the user can fire `hello` and
// `network-monitor` back-to-back and see independent timings + output.
type runState struct {
	Status   runStatus
	Output   string
	Err      string
	StartAt  time.Time
	FinishAt time.Time
}

// Model is the Bubble Tea model for the Claude Code tab.
//
// There is no global "running" / "done" state — each command in m.runs has
// its own status. Helper methods (focused, runningCount, anyDone) collapse
// the map into the high-level facts the View and Help bar need.
type Model struct {
	commands []Command
	cursor   int
	runs     map[string]*runState

	contentWidth  int
	contentHeight int
	vp            viewport.Model
}

// New builds a fresh Model and loads the initial command list from disk.
func New() Model {
	return Model{
		commands: Discover(),
		runs:     map[string]*runState{},
		vp:       viewport.New(0, 0),
	}
}

// Inputting is always false — the Claude tab never owns the keyboard.
// Background runs don't change that: the user is free to tab to other
// panes while jobs are in flight, and RunDoneMsg events are routed to
// this tab regardless of which pane is active.
func (m Model) Inputting() bool { return false }

// Commands exposes the loaded command list for tests.
func (m Model) Commands() []Command { return m.commands }

// Cursor exposes the current highlight index for tests.
func (m Model) Cursor() int { return m.cursor }

// Runs exposes the per-command state map for tests.
func (m Model) Runs() map[string]*runState { return m.runs }

// Output returns the focused command's last output, or "" if the focused
// command has never been run.
func (m Model) Output() string {
	if r := m.focused(); r != nil {
		return r.Output
	}
	return ""
}

// Count reports the number of discovered commands.
func (m Model) Count() int { return len(m.commands) }

// Title is the subtitle shown in the breadcrumb header. It folds the
// per-command run states into a single short line.
func (m Model) Title() string {
	running := m.runningCount()
	switch {
	case running > 1:
		return fmt.Sprintf("%d running", running)
	case running == 1:
		return "running · " + m.runningName()
	case len(m.runs) > 0:
		return fmt.Sprintf("%d command%s · idle", len(m.commands), plural(len(m.commands)))
	default:
		return fmt.Sprintf("%d command%s", len(m.commands), plural(len(m.commands)))
	}
}

// Help returns the key hints shown in the bottom help bar. Scroll/clear
// hints only appear when the focused command has output to act on.
func (m Model) Help() []theme.KeyHint {
	hints := []theme.KeyHint{
		{Key: "↑↓", Label: "navigate"},
		{Key: "enter", Label: "run"},
	}
	if r := m.focused(); r != nil && r.Status == runDone {
		hints = append(hints,
			theme.KeyHint{Key: "space", Label: "scroll"},
			theme.KeyHint{Key: "esc", Label: "clear"},
		)
	}
	return hints
}

// SetRowWidth accepts the main-pane content width from the app.
func (m *Model) SetRowWidth(w int) {
	m.contentWidth = w
	m.resizeViewport()
}

// SetContentHeight accepts the main-pane usable height from the app.
func (m *Model) SetContentHeight(h int) {
	m.contentHeight = h
	m.resizeViewport()
}

// resizeViewport recomputes viewport dimensions from width/height and the
// known chrome above it. Chrome budget: header + blank + COMMANDS divider
// + N command rows + blank + OUTPUT divider + slack for help bar.
func (m *Model) resizeViewport() {
	if m.contentWidth <= 0 {
		return
	}
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

// Update routes RunDoneMsg results back to the right command and dispatches
// keystrokes to the cursor / viewport as appropriate.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RunDoneMsg:
		r := m.runs[msg.Name]
		if r == nil {
			r = &runState{}
			m.runs[msg.Name] = r
		}
		r.Status = runDone
		r.Output = msg.Output
		r.FinishAt = time.Now()
		if msg.Err != nil {
			r.Err = msg.Err.Error()
		} else {
			r.Err = ""
		}
		// Refresh viewport only if the user is still looking at this command.
		if focused := m.focusedName(); focused == msg.Name {
			m.resizeViewport()
			m.vp.SetContent(r.Output)
			m.vp.GotoTop()
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
			m.refreshViewport()
		}
		return m, nil
	case tea.KeyDown:
		if m.cursor < len(m.commands)-1 {
			m.cursor++
			m.refreshViewport()
		}
		return m, nil
	case tea.KeyEnter:
		return m.startRun()
	case tea.KeyEsc:
		// Clear only the focused command's run, not every concurrent run.
		// In-flight runs continue and report back when they finish.
		name := m.focusedName()
		if r := m.runs[name]; r != nil && r.Status == runDone {
			delete(m.runs, name)
			m.vp.SetContent("")
		}
		return m, nil
	}

	// Anything else (space/b/f/j/k/u/d/ctrl+u/ctrl+d) goes to the viewport
	// — but only if the focused command has output worth scrolling.
	if r := m.focused(); r != nil && r.Status == runDone {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

// startRun fires the cursor's command. If that command is already running
// the keystroke is ignored — there's no value in stacking duplicate runs
// of the same command. Other commands' runs are unaffected, so the user
// can move the cursor and Enter on a different command to launch it
// concurrently.
func (m Model) startRun() (tea.Model, tea.Cmd) {
	if len(m.commands) == 0 {
		return m, nil
	}
	c := m.commands[m.cursor]
	if r := m.runs[c.Name]; r != nil && r.Status == runRunning {
		return m, nil
	}
	m.runs[c.Name] = &runState{
		Status:  runRunning,
		StartAt: time.Now(),
	}
	// Clear the viewport since the previous output for this command is
	// being superseded; it will get refilled when RunDoneMsg arrives.
	m.vp.SetContent("")
	return m, RunCommandCmd(c.Name, c.Prompt, c.ExtraArgs...)
}

// refreshViewport repopulates the viewport with the current focused
// command's output. Called after every cursor move so the panel below the
// command list always reflects the highlighted row.
func (m *Model) refreshViewport() {
	r := m.focused()
	if r == nil || r.Status != runDone {
		m.vp.SetContent("")
		return
	}
	m.resizeViewport()
	m.vp.SetContent(r.Output)
	m.vp.GotoTop()
}

// focused returns the runState of the cursor's command, or nil if the
// cursor isn't on anything (empty list) or that command has never run.
func (m Model) focused() *runState {
	if len(m.commands) == 0 {
		return nil
	}
	return m.runs[m.commands[m.cursor].Name]
}

func (m Model) focusedName() string {
	if len(m.commands) == 0 {
		return ""
	}
	return m.commands[m.cursor].Name
}

// runningCount sums the in-flight runs across all commands.
func (m Model) runningCount() int {
	n := 0
	for _, r := range m.runs {
		if r.Status == runRunning {
			n++
		}
	}
	return n
}

// runningName returns the name of one running command (deterministic enough
// for breadcrumb display when exactly one run is active).
func (m Model) runningName() string {
	for _, c := range m.commands {
		if r := m.runs[c.Name]; r != nil && r.Status == runRunning {
			return c.Name
		}
	}
	return ""
}

// View renders the tab content (header, command list, focused output, help bar).
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

	sb.WriteString("\n")
	sb.WriteString(theme.RenderKeyHints(m.Help()))
	return sb.String()
}

// renderHeader summarises tab-wide activity above the command list.
func (m Model) renderHeader() string {
	running := m.runningCount()
	if running == 1 {
		name := m.runningName()
		elapsed := time.Since(m.runs[name].StartAt).Truncate(time.Second)
		return theme.ConnectedDot.Render("● ") +
			theme.Normal.Render("running ") +
			theme.ConnectedText.Render(name) +
			"  " + theme.Dimmed.Render("("+elapsed.String()+")")
	}
	if running > 1 {
		return theme.ConnectedDot.Render("● ") +
			theme.Normal.Render(fmt.Sprintf("%d commands running concurrently", running))
	}
	return theme.Dimmed.Render("pick a command and press ") +
		theme.ConnectedText.Render("enter") +
		theme.Dimmed.Render(" — multiple commands can run in parallel")
}

// renderRow draws one command line plus a per-row status badge so the user
// can see at a glance which runs are live.
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

	if len(c.ExtraArgs) > 0 {
		flags := strings.Join(c.ExtraArgs, " ")
		if hasDangerousFlag(c.ExtraArgs) {
			row += "  " + theme.StatusErr.Render(flags)
		} else {
			row += "  " + theme.Dimmed.Render(flags)
		}
	}
	if badge := m.renderStatusBadge(c.Name); badge != "" {
		row += "  " + badge
	}
	return row
}

// renderStatusBadge produces the trailing per-row status indicator: nothing
// when idle, a live elapsed-time stamp when running, a duration + error
// glyph when done.
func (m Model) renderStatusBadge(name string) string {
	r, ok := m.runs[name]
	if !ok {
		return ""
	}
	switch r.Status {
	case runRunning:
		elapsed := time.Since(r.StartAt).Truncate(time.Second)
		return theme.ConnectedDot.Render("● ") +
			theme.ConnectedText.Render("running "+elapsed.String())
	case runDone:
		dur := r.FinishAt.Sub(r.StartAt).Truncate(time.Millisecond)
		if r.Err != "" {
			return theme.StatusErr.Render("✗ " + dur.String() + " failed")
		}
		return theme.ConnectedDot.Render("✓ ") +
			theme.Dimmed.Render(dur.String())
	}
	return ""
}

func hasDangerousFlag(args []string) bool {
	for _, a := range args {
		if strings.Contains(a, "dangerously") {
			return true
		}
	}
	return false
}

// renderOutput shows the focused command's most recent output. If the
// focused command is mid-run we show a "waiting" notice; if idle we leave
// the output area blank so the rest of the UI can breathe.
func (m Model) renderOutput() string {
	r := m.focused()
	if r == nil {
		return ""
	}
	switch r.Status {
	case runRunning:
		return theme.Dimmed.Render("  waiting for claude…  ") +
			theme.Dimmed.Render("(other commands may also be running — ↑↓ to check)")
	case runDone:
		var sb strings.Builder
		sb.WriteString(theme.Dimmed.Render("── OUTPUT ") +
			theme.Dimmed.Render(strings.Repeat("─", 62)))
		sb.WriteString("\n")
		if r.Err != "" {
			sb.WriteString(theme.StatusErr.Render("  " + r.Err))
			sb.WriteString("\n")
		}
		if r.Output != "" {
			sb.WriteString(m.vp.View())
			sb.WriteString("\n")
			sb.WriteString(m.scrollHint())
		}
		return sb.String()
	}
	return ""
}

// scrollHint summarises whether there's more output above or below the
// current viewport position.
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
