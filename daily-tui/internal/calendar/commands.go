package calendar

import (
	tea "github.com/charmbracelet/bubbletea"
)

// EventsLoadedMsg is delivered when a fetch completes (successfully or not).
type EventsLoadedMsg struct {
	Events []Event
	Err    error
}

// FetchEventsCmd returns a tea.Cmd that calls osascript and parses events.
func FetchEventsCmd(daysAhead int) tea.Cmd {
	return func() tea.Msg {
		raw, err := FetchRaw(daysAhead)
		if err != nil {
			return EventsLoadedMsg{Err: err}
		}
		return EventsLoadedMsg{Events: ParseEvents(raw)}
	}
}
