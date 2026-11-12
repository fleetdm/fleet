package util

import (
	"time"

	"github.com/inconshreveable/log15"
)

// ParsedOrDefaultTime returns time.Parse(layout, value), or time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC) if it failed to parse
func ParsedOrDefaultTime(layouts []string, value string) time.Time {
	defaultTime := time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if value == "" || value == "unknown" {
		return defaultTime
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	log15.Warn("Failed to parse string", "timeformat", layouts, "target string", value)
	return defaultTime
}
