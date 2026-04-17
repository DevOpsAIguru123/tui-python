package calendar

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCache_roundTrip(t *testing.T) {
	cacheBaseDir = t.TempDir()

	day := time.Date(2026, 4, 17, 9, 0, 0, 0, time.Local)
	in := []Event{
		{Calendar: "Work", Title: "Standup", Start: day, End: day.Add(15 * time.Minute)},
	}
	SaveCache(ViewToday, in)

	got, fresh := LoadCache(ViewToday)
	if !fresh {
		t.Fatalf("freshly saved cache should be fresh")
	}
	if len(got.Events) != 1 || got.Events[0].Title != "Standup" {
		t.Errorf("cache round-trip mismatch: %+v", got.Events)
	}
	if got.Age() > time.Second {
		t.Errorf("just-saved entry reported Age=%v", got.Age())
	}
}

func TestCache_staleBeyondTTL(t *testing.T) {
	cacheBaseDir = t.TempDir()

	// Write an entry with a FetchedAt older than the TTL.
	entry := CacheEntry{
		Events:    []Event{{Title: "Old"}},
		FetchedAt: time.Now().Add(-2 * CacheTTL),
	}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(cachePath(ViewToday), data, 0644); err != nil {
		t.Fatal(err)
	}

	got, fresh := LoadCache(ViewToday)
	if fresh {
		t.Error("entry older than TTL should not be reported fresh")
	}
	if len(got.Events) != 1 {
		t.Error("stale entries should still load (for display while refresh runs)")
	}
}

func TestCache_missingFile(t *testing.T) {
	cacheBaseDir = t.TempDir()
	if _, fresh := LoadCache(ViewMonth); fresh {
		t.Error("missing cache file should not be fresh")
	}
}

func TestCache_perViewIsolation(t *testing.T) {
	cacheBaseDir = t.TempDir()

	SaveCache(ViewToday, []Event{{Title: "today-only"}})
	SaveCache(ViewWeek, []Event{{Title: "week-only"}})

	today, _ := LoadCache(ViewToday)
	week, _ := LoadCache(ViewWeek)

	if len(today.Events) != 1 || today.Events[0].Title != "today-only" {
		t.Errorf("today cache wrong: %+v", today.Events)
	}
	if len(week.Events) != 1 || week.Events[0].Title != "week-only" {
		t.Errorf("week cache wrong: %+v", week.Events)
	}
}
