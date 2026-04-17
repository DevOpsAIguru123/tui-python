package calendar

import "time"

// Event is a single calendar entry surfaced from macOS Calendar.app.
type Event struct {
	Calendar string
	Title    string
	Start    time.Time
	End      time.Time
	Location string
}

// IsAllDay reports whether the event spans a full day (midnight→midnight).
func (e Event) IsAllDay() bool {
	return e.Start.Hour() == 0 && e.Start.Minute() == 0 &&
		e.End.Sub(e.Start) >= 24*time.Hour
}
