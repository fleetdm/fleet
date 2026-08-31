package sentinelone

import (
	"strconv"
	"strings"
	"time"
)

// timeLayouts are tried in order against a raw sentinelctl timestamp. macOS
// emits a locale-influenced short form ("6/1/26, 10:09:51 AM"); the rest cover
// variants seen on other platforms.
var timeLayouts = []string{
	"1/2/06, 3:04:05 PM",
	"1/2/2006, 3:04:05 PM",
	"1/2/06 3:04:05 PM",
	"1/2/2006 3:04:05 PM",
	"2006-01-02 15:04:05",
	time.RFC3339,
}

// toUnixEpoch converts a raw sentinelctl timestamp to Unix epoch seconds as a
// decimal string. A value without an explicit zone is read in the host's local
// zone, which is what sentinelctl reports in.
func toUnixEpoch(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return strconv.FormatInt(t.Unix(), 10), true
		}
	}
	if t, ok := parseShortDateTime(s); ok {
		return strconv.FormatInt(t.Unix(), 10), true
	}
	return "", false
}

// parseShortDateTime parses the "M/D/YY[YY][,] H:MM[:SS] [AM|PM]" form macOS
// sentinelctl emits. It splits on every non-alphanumeric run, so it tolerates
// the non-breaking spaces, missing commas and irregular padding that defeat
// the strict layouts above.
//
// The field order is assumed to be month-first. On a host whose locale emits
// D/M/Y, a day past the 12th fails the month range check and the column is
// cleared instead of storing a transposed date; an earlier day is
// indistinguishable from M/D/Y and is read as such.
func parseShortDateTime(s string) (time.Time, bool) {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	})
	// month, day, year, hour, minute, [second], [am|pm]
	if len(fields) < 5 {
		return time.Time{}, false
	}
	month, monthOK := atoiRange(fields[0], 1, 12)
	day, dayOK := atoiRange(fields[1], 1, 31)
	year, yearOK := atoiRange(fields[2], 0, 9999)
	hour, hourOK := atoiRange(fields[3], 0, 23)
	minute, minuteOK := atoiRange(fields[4], 0, 59)
	if !monthOK || !dayOK || !yearOK || !hourOK || !minuteOK {
		return time.Time{}, false
	}
	if year < 100 {
		year += 2000
	}

	next := 5
	second := 0
	if next < len(fields) {
		if sec, ok := atoiRange(fields[next], 0, 60); ok {
			second = sec
			next++
		}
	}
	if next < len(fields) {
		switch strings.ToUpper(fields[next]) {
		case "AM":
			if hour == 12 {
				hour = 0
			}
		case "PM":
			if hour < 12 {
				hour += 12
			}
		}
	}
	if hour > 23 {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local), true
}

// atoiRange parses s as a decimal integer within [lo, hi].
func atoiRange(s string, lo, hi int) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n < lo || n > hi {
		return 0, false
	}
	return n, true
}
