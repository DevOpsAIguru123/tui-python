package calendar

// View selects the time window surfaced by the Calendar tab.
type View int

const (
	ViewToday View = iota
	ViewWeek
	ViewMonth
)

// AllViews lists the views in the order they appear in the tab header,
// which is also the order of the 1 / 2 / 3 shortcut keys.
var AllViews = []View{ViewToday, ViewWeek, ViewMonth}

// Days returns how many days ahead from today 00:00 the view covers.
func (v View) Days() int {
	switch v {
	case ViewToday:
		return 1
	case ViewWeek:
		return 7
	case ViewMonth:
		return 30
	}
	return 7
}

// Label returns the human-readable name shown in the tab header.
func (v View) Label() string {
	switch v {
	case ViewToday:
		return "Today"
	case ViewWeek:
		return "This week"
	case ViewMonth:
		return "This month"
	}
	return ""
}

// Slug returns the cache-file-safe identifier for the view.
func (v View) Slug() string {
	switch v {
	case ViewToday:
		return "today"
	case ViewWeek:
		return "week"
	case ViewMonth:
		return "month"
	}
	return "unknown"
}
