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
}

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

// clipLocation shortens an event location string to at most max runes,
// appending an ellipsis if anything was cut. The calendar row doesn't need
// the full address — a recognisable prefix is enough and leaves room for
// the event title and calendar badge on the same line.
func clipLocation(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
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
		sb.WriteString(renderEventLine(e, i == m.cursor))
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
		// Clip long locations so a single event can't push the row past the
		// pane's right edge (e.g. "@ Austin Central Library, Austin, TX").
		line += theme.Dimmed.Render("  @ " + clipLocation(e.Location, 28))
	}
	if e.Calendar != "" {
		line += "  " + theme.Badge.Render(e.Calendar)
	}
	if selected {
		return theme.Selected.Render("▸ " + line)
	}
	return theme.Normal.Render("  " + line)
}
