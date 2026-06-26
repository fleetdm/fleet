package processes

import (
	"regexp"
	"sort"
	"strings"
)

// logChannelCap is the per-channel in-memory ring size (oldest dropped).
const logChannelCap = 50_000

// ring is a fixed-capacity circular buffer of LogEntry. Push is O(1); the
// oldest entry is overwritten once full.
type ring struct {
	data  []LogEntry
	start int // index of the oldest element
	size  int
}

func newRing(capacity int) *ring {
	return &ring{data: make([]LogEntry, capacity)}
}

func (r *ring) push(e LogEntry) {
	n := len(r.data)
	if r.size < n {
		r.data[(r.start+r.size)%n] = e
		r.size++
		return
	}
	r.data[r.start] = e
	r.start = (r.start + 1) % n
}

// snapshot returns the entries oldest→newest (a copy; safe to use unlocked).
func (r *ring) snapshot() []LogEntry {
	out := make([]LogEntry, r.size)
	n := len(r.data)
	for i := 0; i < r.size; i++ {
		out[i] = r.data[(r.start+i)%n]
	}
	return out
}

// filterLogWindow applies the source/time/level/search filters to a snapshot
// of channel→entries (each oldest→newest) and returns the windowed result.
// Pure (no locking, no Manager state) so it's directly testable.
//
//   - source "all" scans every channel; otherwise just the named one.
//   - an entry with no detected level is treated as "info".
//   - an empty levels set shows nothing (every chip toggled off).
//   - search wrapped in /.../ is a regex (invalid regex disables filtering),
//     otherwise a case-insensitive substring.
//   - maxLines caps the returned entries to the newest N (counts are pre-cap).
func filterLogWindow(channels map[string][]LogEntry, source string, sinceMS uint64, levels []string, search *string, maxLines *int) LogWindow {
	levelSet := map[string]bool{}
	for _, l := range levels {
		levelSet[l] = true
	}

	var re *regexp.Regexp
	isRegex := false
	searchLower := ""
	hasSearchLower := false
	if search != nil {
		s := *search
		if len(s) >= 3 && strings.HasPrefix(s, "/") && strings.HasSuffix(s, "/") {
			isRegex = true
			if r, err := regexp.Compile(s[1 : len(s)-1]); err == nil {
				re = r
			}
		} else {
			searchLower = strings.ToLower(s)
			hasSearchLower = true
		}
	}
	_ = isRegex

	var keys []string
	if source == "all" {
		for k := range channels {
			keys = append(keys, k)
		}
	} else if _, ok := channels[source]; ok {
		keys = []string{source}
	}

	var entries []LogEntry
	total, warn, errc := 0, 0, 0
	for _, k := range keys {
		for _, e := range channels[k] {
			if e.TsMS < sinceMS {
				continue
			}
			lvl := "info"
			if e.Level != nil {
				lvl = *e.Level
			}
			if !levelSet[lvl] {
				continue
			}
			if re != nil {
				if !re.MatchString(e.Message) {
					continue
				}
			} else if hasSearchLower {
				if !strings.Contains(strings.ToLower(e.Message), searchLower) {
					continue
				}
			}
			total++
			switch lvl {
			case "warn":
				warn++
			case "error":
				errc++
			}
			entries = append(entries, e)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].TsMS < entries[j].TsMS })
	if maxLines != nil && len(entries) > *maxLines {
		entries = entries[len(entries)-*maxLines:]
	}

	return LogWindow{Entries: entries, TotalInWindow: total, WarnCount: warn, ErrorCount: errc}
}
