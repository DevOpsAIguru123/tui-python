package calendar

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// EventsLoadedMsg is delivered when a fetch completes (successfully or not).
// View tags which time-range the result belongs to, so stale responses for a
// view the user has since left can be discarded.
type EventsLoadedMsg struct {
	View      View
	Events    []Event
	FetchedAt time.Time
	Err       error
}

// FetchEventsCmd runs osascript for v, writes the result to the on-disk
// cache, and wraps it in an EventsLoadedMsg.
func FetchEventsCmd(v View) tea.Cmd {
	return func() tea.Msg {
		raw, err := FetchRaw(v.Days())
		if err != nil {
			return EventsLoadedMsg{View: v, Err: err}
		}
		events := ParseEvents(raw)
		SaveCache(v, events)
		return EventsLoadedMsg{View: v, Events: events, FetchedAt: time.Now()}
	}
}
