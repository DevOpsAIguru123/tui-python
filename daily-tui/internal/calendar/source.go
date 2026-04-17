package calendar

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// applescriptTmpl produces one tab-separated line per event:
//
//	calendar \t title \t start(ISO local) \t end(ISO local) \t location
//
// %d is replaced with the number of days to look ahead from today 00:00.
const applescriptTmpl = `on padN(n, w)
	set s to n as string
	repeat while length of s < w
		set s to "0" & s
	end repeat
	return s
end padN

on isoDate(d)
	return (year of d as string) & "-" & my padN((month of d as integer), 2) & "-" & my padN(day of d, 2) & "T" & my padN(hours of d, 2) & ":" & my padN(minutes of d, 2) & ":" & my padN(seconds of d, 2)
end isoDate

tell application "Calendar"
	set startDate to current date
	set time of startDate to 0
	set endDate to startDate + (%d * days)
	set output to ""
	repeat with c in calendars
		set calName to name of c
		try
			tell c
				set evts to (every event whose start date is greater than or equal to startDate and start date is less than endDate)
				repeat with e in evts
					set loc to location of e
					if loc is missing value then set loc to ""
					set output to output & calName & tab & (summary of e) & tab & my isoDate(start date of e) & tab & my isoDate(end date of e) & tab & loc & linefeed
				end repeat
			end tell
		end try
	end repeat
	return output
end tell
`

// Supported reports whether this platform can fetch live events.
// Only macOS has Calendar.app + osascript.
func Supported() bool { return runtime.GOOS == "darwin" }

// FetchRaw runs osascript and returns its stdout.
// daysAhead clamped to [1, 365].
func FetchRaw(daysAhead int) (string, error) {
	if !Supported() {
		return "", fmt.Errorf("calendar: unsupported platform %s (macOS only)", runtime.GOOS)
	}
	if daysAhead < 1 {
		daysAhead = 1
	}
	if daysAhead > 365 {
		daysAhead = 365
	}
	script := fmt.Sprintf(applescriptTmpl, daysAhead)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return string(out), nil
}

// ParseEvents parses the tab-separated output of the AppleScript into Events,
// sorted by start time ascending. Malformed lines are skipped.
func ParseEvents(raw string) []Event {
	var out []Event
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		start, err := time.ParseInLocation("2006-01-02T15:04:05", parts[2], time.Local)
		if err != nil {
			continue
		}
		end, err := time.ParseInLocation("2006-01-02T15:04:05", parts[3], time.Local)
		if err != nil {
			continue
		}
		loc := ""
		if len(parts) >= 5 {
			loc = parts[4]
		}
		out = append(out, Event{
			Calendar: parts[0],
			Title:    parts[1],
			Start:    start,
			End:      end,
			Location: loc,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}
