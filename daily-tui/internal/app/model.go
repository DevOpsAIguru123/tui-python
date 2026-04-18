package app

import (
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/calendar"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/portfolio"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/tux"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabWifi      = 0
	tabTodo      = 1
	tabCalendar  = 2
	tabPortfolio = 3
)

// Tab glyphs displayed next to each sidebar entry. They serve both as visual
// shorthand (◆, ●, ◢, ♦) and as a hint at the active tab via highlighting.
type tabSpec struct {
	Name  string
	Glyph string
}

var tabs = []tabSpec{
	{Name: "WiFi", Glyph: "◆"},
	{Name: "Todo", Glyph: "●"},
	{Name: "Calendar", Glyph: "◈"},
	{Name: "Portfolio", Glyph: "♦"},
}

const sidebarWidth = 20

// tickMsg drives the clock in the header. We tick every 30s so the displayed
// time stays within the minute without burning CPU on sub-second repaints.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Model is the root Bubble Tea model.
type Model struct {
	activeTab int
	wifi      wifi.Model
	todo      todo.Model
	calendar  calendar.Model
	portfolio portfolio.Model
	width     int
	height    int
	now       time.Time
	version   string
}

// New creates the root AppModel.
func New(wm wifi.Model, tm todo.Model, cm calendar.Model, pm portfolio.Model, version string) Model {
	return Model{
		wifi:      wm,
		todo:      tm,
		calendar:  cm,
		portfolio: pm,
		version:   version,
		now:       time.Now(),
	}
}

// ActiveTab returns the index of the currently active tab.
func (m Model) ActiveTab() int { return m.activeTab }

// Init starts all child models and kicks off the clock tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.wifi.Init(),
		m.todo.Init(),
		m.calendar.Init(),
		m.portfolio.Init(),
		tickCmd(),
	)
}

// Update handles messages: tab-switch keys are consumed here; all others
// are forwarded to the active child.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		content := m.mainWidth() - 6
		m.wifi.SetRowWidth(content)
		m.todo.SetRowWidth(content)
		m.calendar.SetRowWidth(content)
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickCmd()

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// Only intercept navigation keys when the active child is not in an input mode.
		// If the user is typing a password or a todo, all keys belong to the child.
		childInputting := (m.activeTab == tabWifi && m.wifi.Inputting()) ||
			(m.activeTab == tabTodo && m.todo.Inputting()) ||
			(m.activeTab == tabCalendar && m.calendar.Inputting()) ||
			(m.activeTab == tabPortfolio && m.portfolio.Inputting())
		if !childInputting {
			if msg.Type == tea.KeyRunes && string(msg.Runes) == "q" {
				return m, tea.Quit
			}
			if msg.Type == tea.KeyTab {
				m.activeTab = (m.activeTab + 1) % len(tabs)
				return m, nil
			}
			if msg.Type == tea.KeyRunes {
				// Calendar and Portfolio claim the number keys for their own
				// in-tab navigation (Calendar: Today/Week/Month, Portfolio:
				// Holdings/Watchlist). Only treat digits as tab-switches when
				// the active tab doesn't need them.
				if m.activeTab != tabCalendar && m.activeTab != tabPortfolio {
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
					case "4":
						m.activeTab = tabPortfolio
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
	case tabPortfolio:
		updated, cmd := m.portfolio.Update(msg)
		m.portfolio = updated.(portfolio.Model)
		return m, cmd
	}
	return m, nil
}

// View renders the full TUI: sidebar + (header, content, help) column.
func (m Model) View() string {
	sidebar := m.renderSidebar()
	main := m.renderMain()
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

// ---- sidebar ----

func (m Model) renderSidebar() string {
	var sb strings.Builder

	sb.WriteString(theme.Brand.Render("daily-tui"))
	sb.WriteString("\n")
	sb.WriteString(theme.BrandSub.Render(m.brandSubtitle()))
	sb.WriteString("\n\n")

	sb.WriteString(tux.Render())
	sb.WriteString("\n\n")

	sb.WriteString(theme.SidebarLabel.Render("TABS"))
	sb.WriteString("\n")
	for i, t := range tabs {
		sb.WriteString(m.renderSidebarTab(i, t))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	// Pad the remaining vertical space so HOST always sits near the bottom.
	hostBlock := m.renderSidebarHost()
	// Compose the header + padding + host as a single block so the sidebar
	// style (below) can lay it out consistently.
	top := sb.String()

	// A single right-edge border acts as the visual divider between the
	// sidebar and the main pane. Using a border (rather than a column of
	// pipe characters) lets lipgloss stretch it to match the taller of the
	// two panes automatically.
	panel := lipgloss.NewStyle().
		Width(sidebarWidth).
		Padding(1, 2).
		Foreground(theme.Text).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(theme.Surface1).
		Render(top + "\n" + hostBlock)
	return panel
}

func (m Model) brandSubtitle() string {
	v := formatVersion(m.version)
	iface := m.wifi.Iface()
	if iface == "" {
		return v
	}
	return v + " · " + iface
}

// formatVersion decides how to render a build-time version string. Semver-ish
// values get a "v" prefix ("0.3.1" → "v0.3.1"); anything else (git SHA,
// "dev", etc.) is passed through verbatim so we don't emit e.g. "vd9870be".
func formatVersion(v string) string {
	if v == "" {
		return "dev"
	}
	if v == "dev" {
		return v
	}
	first := v[0]
	if first >= '0' && first <= '9' {
		return "v" + v
	}
	if first == 'v' && len(v) > 1 {
		next := v[1]
		if next >= '0' && next <= '9' {
			return v
		}
	}
	return v
}

func (m Model) renderSidebarTab(i int, t tabSpec) string {
	active := i == m.activeTab

	glyph := t.Glyph
	if active {
		glyph = theme.TabBulletActive.Render(glyph)
	} else {
		glyph = theme.TabBulletIdle.Render(glyph)
	}

	name := t.Name
	if active {
		name = theme.TabRowActive.Render(name)
	} else {
		name = theme.TabRowInactive.Render(name)
	}

	return glyph + " " + name
}

func (m Model) renderSidebarHost() string {
	var sb strings.Builder
	sb.WriteString(theme.SidebarLabel.Render("HOST"))
	sb.WriteString("\n")
	sb.WriteString(theme.SidebarPromptSym.Render("$ whoami"))
	sb.WriteString("\n")
	sb.WriteString(theme.SidebarPromptVal.Render(currentUser()))
	sb.WriteString("\n")

	iface := m.wifi.Iface()
	if iface != "" {
		sb.WriteString(theme.SidebarPromptSym.Render("$ iface"))
		sb.WriteString("\n")
		sb.WriteString(theme.SidebarPromptVal.Render(iface))
		sb.WriteString("\n")
	}

	// Global keys live here (always visible, never overflow the main pane).
	sb.WriteString("\n")
	sb.WriteString(theme.SidebarLabel.Render("KEYS"))
	sb.WriteString("\n")
	sb.WriteString(theme.KeyDesc.Render("tab ") + theme.Dimmed.Render("switch"))
	sb.WriteString("\n")
	sb.WriteString(theme.KeyDesc.Render("q   ") + theme.Dimmed.Render("quit"))
	sb.WriteString("\n")
	return sb.String()
}

func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	return "user"
}

// ---- main pane ----

func (m Model) renderMain() string {
	width := m.mainWidth()

	header := m.renderHeader(width)
	content := m.activeView()

	// Only pad; do NOT set Width. A fixed Width forces lipgloss to wrap any
	// overflowing line (e.g. the help keycap bar on narrower terminals),
	// which made the help bar break into a 2-row grid.
	padded := lipgloss.NewStyle().
		Padding(1, 2, 0, 2).
		Render(header + "\n\n" + content)

	return padded
}

// mainWidth reports the usable content width of the main pane. It accounts
// for the sidebar itself (sidebarWidth), the sidebar's right border column
// (+1), and the main pane's left+right padding (+4).
func (m Model) mainWidth() int {
	const chrome = sidebarWidth + 1 + 4
	if m.width > chrome+20 {
		return m.width - chrome
	}
	return 90
}

func (m Model) renderHeader(width int) string {
	caret := theme.CrumbCaret.Render("›")
	tab := theme.CrumbActive.Render(tabs[m.activeTab].Name)
	sep := theme.CrumbSep.Render(" · ")
	title := theme.Crumb.Render(m.activeTitle())

	left := caret + " " + tab + sep + title
	right := theme.Clock.Render(m.now.Format("3:04 PM"))

	pad := width - 4 - visibleWidth(left) - visibleWidth(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func (m Model) activeTitle() string {
	switch m.activeTab {
	case tabWifi:
		return m.wifi.Title()
	case tabTodo:
		return m.todo.Title()
	case tabCalendar:
		return m.calendar.Title()
	case tabPortfolio:
		return m.portfolio.Title()
	}
	return ""
}

func (m Model) activeView() string {
	switch m.activeTab {
	case tabWifi:
		return m.wifi.View()
	case tabTodo:
		return m.todo.View()
	case tabCalendar:
		return m.calendar.View()
	case tabPortfolio:
		return m.portfolio.View()
	}
	return ""
}

// ---- helpers ----

func visibleWidth(s string) int {
	out := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		out++
	}
	return out
}
