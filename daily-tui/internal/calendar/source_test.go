package calendar

import (
	"testing"
	"time"
)

func TestParseEvents_basic(t *testing.T) {
	raw := "Work\tStandup\t2026-04-17T09:00:00\t2026-04-17T09:15:00\t\n" +
		"Personal\tDentist\t2026-04-17T14:00:00\t2026-04-17T15:00:00\t123 Main St\n"
	got := ParseEvents(raw)
	if len(got) != 2 {
		t.Fatalf("want 2 events, got %d", len(got))
	}
	if got[0].Title != "Standup" || got[0].Calendar != "Work" {
		t.Errorf("event 0 mismatch: %+v", got[0])
	}
	if got[1].Location != "123 Main St" {
		t.Errorf("event 1 location: want '123 Main St', got %q", got[1].Location)
	}
	if !got[0].Start.Before(got[1].Start) {
		t.Error("events should be sorted by start time ascending")
	}
}

func TestParseEvents_sortsByStart(t *testing.T) {
	// Intentionally unordered input
	raw := "A\tLater\t2026-04-17T15:00:00\t2026-04-17T16:00:00\t\n" +
		"A\tEarlier\t2026-04-17T09:00:00\t2026-04-17T10:00:00\t\n"
	got := ParseEvents(raw)
	if len(got) != 2 || got[0].Title != "Earlier" || got[1].Title != "Later" {
		t.Errorf("not sorted ascending: %+v", got)
	}
}

func TestParseEvents_skipsMalformed(t *testing.T) {
	raw := "broken line with no tabs\n" +
		"Work\tOnly\t2026-04-17T09:00:00\t2026-04-17T10:00:00\t\n" +
		"Work\tBadStart\tNOT-A-DATE\t2026-04-17T10:00:00\t\n" +
		"\n" // empty line
	got := ParseEvents(raw)
	if len(got) != 1 || got[0].Title != "Only" {
		t.Errorf("want 1 surviving event named 'Only', got %+v", got)
	}
}

func TestParseEvents_missingLocationColumn(t *testing.T) {
	// 4 fields only — parser should still accept it with empty Location.
	raw := "Work\tNoLoc\t2026-04-17T09:00:00\t2026-04-17T10:00:00\n"
	got := ParseEvents(raw)
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].Location != "" {
		t.Errorf("want empty location, got %q", got[0].Location)
	}
}

func TestEvent_IsAllDay(t *testing.T) {
	day := time.Date(2026, 4, 17, 0, 0, 0, 0, time.Local)
	if !(Event{Start: day, End: day.Add(24 * time.Hour)}).IsAllDay() {
		t.Error("midnight→next midnight should be all-day")
	}
	noon := time.Date(2026, 4, 17, 12, 0, 0, 0, time.Local)
	if (Event{Start: noon, End: noon.Add(time.Hour)}).IsAllDay() {
		t.Error("noon→1pm should not be all-day")
	}
}
