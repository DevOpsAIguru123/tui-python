package calendar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CacheTTL bounds how long a cache entry is considered fresh.
// A cached view older than this still loads, but is marked stale and
// a background refresh is triggered.
const CacheTTL = 5 * time.Minute

// cacheBaseDir is the directory that holds per-view JSON cache files.
// Overridable from tests.
var cacheBaseDir = filepath.Join(os.Getenv("HOME"), ".cache", "daily-tui")

// CacheEntry is the on-disk shape for a cached view.
type CacheEntry struct {
	Events    []Event   `json:"events"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Age reports how long ago the entry was fetched.
func (e CacheEntry) Age() time.Duration { return time.Since(e.FetchedAt) }

// Fresh reports whether the entry is within CacheTTL.
func (e CacheEntry) Fresh() bool { return e.Age() < CacheTTL }

func cachePath(v View) string {
	return filepath.Join(cacheBaseDir, "calendar-"+v.Slug()+".json")
}

// LoadCache reads the cache entry for v. The second return is true when the
// entry exists and is within CacheTTL. A stale entry is still returned so the
// UI can show it while a fresh fetch runs in the background.
func LoadCache(v View) (CacheEntry, bool) {
	data, err := os.ReadFile(cachePath(v))
	if err != nil {
		return CacheEntry{}, false
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return CacheEntry{}, false
	}
	return entry, entry.Fresh()
}

// SaveCache writes events to disk for v with the current timestamp.
// Errors are ignored — caching is a best-effort optimisation.
func SaveCache(v View, events []Event) {
	if err := os.MkdirAll(cacheBaseDir, 0755); err != nil {
		return
	}
	entry := CacheEntry{Events: events, FetchedAt: time.Now()}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cachePath(v), data, 0644)
}
