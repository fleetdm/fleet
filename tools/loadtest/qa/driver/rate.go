package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// rate is an average number of events per second.
type rate float64

func (r rate) perDay() float64 { return float64(r) * 86400 }

func (r rate) String() string {
	d := r.perDay()
	switch {
	case d >= 10:
		return fmt.Sprintf("%.0f/day", d)
	case d >= 1:
		return fmt.Sprintf("%.1f/day", d)
	default:
		return fmt.Sprintf("%.2f/day", d)
	}
}

var rateUnits = map[string]time.Duration{
	"s": time.Second, "sec": time.Second,
	"min": time.Minute, "m": time.Minute,
	"h": time.Hour, "hour": time.Hour,
	"day": 24 * time.Hour, "d": 24 * time.Hour,
	"week": 7 * 24 * time.Hour,
}

// parseRate reads "40/day", "3/h" or "0.5/min".
func parseRate(s string) (rate, error) {
	num, unit, ok := strings.Cut(strings.TrimSpace(s), "/")
	if !ok {
		return 0, fmt.Errorf("rate %q: want <number>/<unit>, e.g. 40/day", s)
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("rate %q: bad number", s)
	}
	per, ok := rateUnits[strings.TrimSpace(unit)]
	if !ok {
		return 0, fmt.Errorf("rate %q: unknown unit %q (s, min, h, day, week)", s, unit)
	}
	return rate(n / per.Seconds()), nil
}
