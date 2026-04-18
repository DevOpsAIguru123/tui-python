package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the Calendar tab.
type Model struct {
	view      View
	events    []Event
	cursor    int
	loading   bool
	err       error
	fetchedAt time.Time
	rowWidth  int
}

// SetRowWidth receives the main-pane content width from the app so the tab
// can clip long event lines before they overflow the right edge.
func (m *Model) SetRowWidth(w int) { m.rowWidth = w }

// New creates a Calendar model. Fetch is kicked off by Init.
// If a fresh on-disk cache exists for the default view, its events are
// hydrated synchronously so the first paint shows data immediately.
func New() Model {
	m := Model{view: ViewToday}
	if entry, fresh := LoadCache(m.view); fresh {
		m.events = entry.Events
		m.fetchedAt = entry.FetchedAt
	} else {
		m.loading = Supported()
	}
	return m
}

// Inputting reports whether the tab owns all keys (none for read-only view).
func (m Model) Inputting() bool { return false }

// Events returns the currently loaded events (used by tests).
func (m Model) Events() []Event { return m.events }

// CurrentView returns the currently selected time-range filter.
func (m Model) CurrentView() View { return m.view }

// Count returns the number of loaded events (sidebar pill badge).
func (m Model) Count() int { return len(m.events) }

// Title is the subtitle shown in the breadcrumb header. It reflects the
// currently-selected view filter and the number of events in it, so the
// header doubles as a status readout.
func (m Model) Title() string {
	label := strings.ToLower(m.view.Label())
	if m.loading {
		return label + " · loading…"
	}
	if len(m.events) == 0 {
		return label + " · empty"
	}
	return fmt.Sprintf("%s · %d event%s", label, len(m.events), plural(len(m.events)))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Help returns the key hints rendered by the help bar.
func (m Model) Help() []theme.KeyHint {
	return []theme.KeyHint{
		{Key: "↑↓", Label: "navigate"},
		{Key: "1/2/3", Label: "view"},
		{Key: "r", Label: "refresh"},
	}
}

// Init kicks off the initial fetch (or returns nil on unsupported OS).
func (m Model) Init() tea.Cmd {
	if !Supported() {
		return nil
	}
	return FetchEventsCmd(m.view)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case EventsLoadedMsg:
		// Ignore responses for a view the user has since switched away from.
		if msg.View != m.view {
			return m, nil
		}
		m.loading = false
		m.err = msg.Err
		if msg.Err == nil {
			m.events = msg.Events
			m.fetchedAt = msg.FetchedAt
		}
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
			switch string(msg.Runes) {
			case "1":
				return m.switchView(ViewToday)
			case "2":
				return m.switchView(ViewWeek)
			case "3":
				return m.switchView(ViewMonth)
			case "r":
				if Supported() {
					m.loading = len(m.events) == 0
					m.err = nil
					return m, FetchEventsCmd(m.view)
				}
			}
		}
	}
	return m, nil
}

// switchView changes the active time-range filter. A fresh cache entry is
// hydrated synchronously; a background refresh is always kicked off so the
// user sees the latest events even when the cache hit was instant.
func (m Model) switchView(v View) (tea.Model, tea.Cmd) {
	if v == m.view {
		return m, nil
	}
	m.view = v
	m.cursor = 0
	m.err = nil
	if entry, fresh := LoadCache(v); fresh {
		m.events = entry.Events
		m.fetchedAt = entry.FetchedAt
		m.loading = false
	} else {
		m.events = nil
		m.fetchedAt = time.Time{}
		m.loading = Supported()
	}
	if !Supported() {
		return m, nil
	}
	return m, FetchEventsCmd(m.view)
}

// View renders the Calendar tab.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(m.renderViewHeader())
	sb.WriteString("\n\n")

	// Only show the unsupported-platform notice when there's nothing else to
	// show. Tests inject pre-loaded events on non-darwin hosts and still
	// expect the event list to render.
	if !Supported() && len(m.events) == 0 {
		sb.WriteString(theme.StatusErr.Render("Calendar tab is macOS-only (uses Calendar.app via osascript)."))
		sb.WriteString("\n")
		return sb.String()
	}

	if m.loading {
		sb.WriteString(theme.Dimmed.Render("  Loading events..."))
		sb.WriteString("\n\n")
		sb.WriteString(theme.RenderKeyHints(m.Help()))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(theme.StatusErr.Render("Failed to load: " + m.err.Error()))
		sb.WriteString("\n")
		sb.WriteString(theme.Dimmed.Render("  First run may need to grant Terminal access to Calendar in System Settings → Privacy."))
		sb.WriteString("\n\n")
		sb.WriteString(theme.RenderKeyHints([]theme.KeyHint{
			{Key: "1/2/3", Label: "view"},
			{Key: "r", Label: "retry"},
		}))
		return sb.String()
	}

	if len(m.events) == 0 {
		sb.WriteString(theme.Dimmed.Render("  No events in " + strings.ToLower(m.view.Label())))
		sb.WriteString("\n\n")
		sb.WriteString(theme.RenderKeyHints(m.Help()))
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
		sb.WriteString(renderEventLine(e, i == m.cursor, m.rowWidth))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.RenderKeyHints(m.Help()))
	return sb.String()
}

// renderViewHeader draws the view filter row and a "cached X ago" hint
// when a background refresh might still be in flight.
func (m Model) renderViewHeader() string {
	var parts []string
	for _, v := range AllViews {
		label := v.Label()
		if v == m.view {
			parts = append(parts, theme.ActiveTab.Render(label))
		} else {
			parts = append(parts, theme.InactiveTab.Render(label))
		}
	}
	header := strings.Join(parts, " ")
	if hint := m.cacheHint(); hint != "" {
		header += "   " + theme.Dimmed.Render(hint)
	}
	return header
}

// cacheHint returns "cached X ago" when the currently-shown events came from
// a cache and a background refresh is still running or just completed.
func (m Model) cacheHint() string {
	if m.fetchedAt.IsZero() {
		return ""
	}
	age := time.Since(m.fetchedAt)
	if age < 2*time.Second {
		return ""
	}
	return "cached " + humanAge(age) + " ago"
}

func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// renderEventLine renders one agenda row. The pane's usable width (rowWidth)
// is used to clip the overall line so the trailing calendar badge never gets
// shoved off the right edge when title + location add up to a long string.
// When rowWidth is 0 (not yet sized) the line flows at natural width.
func renderEventLine(e Event, selected bool, rowWidth int) string {
	var when string
	if e.IsAllDay() {
		when = "all-day"
	} else {
		when = fmt.Sprintf("%s–%s",
			e.Start.Format("3:04pm"),
			e.End.Format("3:04pm"))
	}
	prefix := "  "
	if selected {
		prefix = theme.Selected.Render("▸ ")
	}

	titleStyle := theme.Normal
	if selected {
		titleStyle = theme.Selected
	}

	// Plan a width budget: prefix + timeCol + title + (loc) + (badge) ≤ rowWidth.
	timeCol := fmt.Sprintf("%-13s ", when)
	budget := rowWidth
	if budget <= 0 {
		budget = 1000 // effectively unbounded
	}
	used := 2 + len(timeCol) // prefix + time col

	var locPart, badgePart string
	if e.Calendar != "" {
		b := "  " + e.Calendar
		if used+runeLen(e.Title)+runeLen(b) <= budget {
			badgePart = "  " + theme.Badge.Render(e.Calendar)
		}
	}
	if e.Location != "" {
		remaining := budget - used - runeLen(e.Title) - visibleLen(badgePart)
		if remaining > 6 {
			loc := "  @ " + e.Location
			if runeLen(loc) > remaining {
				loc = "  @ " + clip(e.Location, remaining-4)
			}
			locPart = theme.Dimmed.Render(loc)
		}
	}

	title := e.Title
	titleBudget := budget - used - visibleLen(locPart) - visibleLen(badgePart)
	if titleBudget > 0 && runeLen(title) > titleBudget {
		title = clip(title, titleBudget)
	}

	return prefix + theme.Dimmed.Render(timeCol) + titleStyle.Render(title) + locPart + badgePart
}

func runeLen(s string) int { return len([]rune(s)) }

// visibleLen strips ANSI escape sequences and returns the rune count; used
// when measuring already-styled fragments against a rendered width budget.
func visibleLen(s string) int {
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

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
