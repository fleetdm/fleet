package main

import (
	"errors"
	"fmt"
	"time"
)

// curve shapes an action's rate over the week. factor averages 1 over a week, so a spec's rate per day stays its
// average day; max is an upper bound of factor, for thinning.
type curve interface {
	factor(t time.Time) float64
	max() float64
}

// curveConfig describes the admin day. The defaults are an assumption until a profile says otherwise.
type curveConfig struct {
	Zones     []string `yaml:"zones"`      // business hours count in each zone, weighted equally
	Open      int      `yaml:"open"`       // local hour business starts
	Close     int      `yaml:"close"`      // local hour business ends
	OffHours  float64  `yaml:"off_hours"`  // weekday activity outside business hours, relative to business hours
	Weekend   float64  `yaml:"weekend"`    // weekend activity, relative to business hours
	BusyHours []int    `yaml:"busy_hours"` // weekday local hours in the first zone that run hot
	BusyX     float64  `yaml:"busy_x"`     // a busy hour runs at this multiple of the weekday hourly average
}

func defaultCurveConfig() curveConfig {
	return curveConfig{
		Zones:     []string{"America/New_York", "America/Chicago", "America/Los_Angeles"},
		Open:      9,
		Close:     17,
		OffHours:  0.05,
		Weekend:   0.1,
		BusyHours: []int{10, 14},
		BusyX:     3,
	}
}

// curves builds the named curves from one config.
func curves(cfg curveConfig) (map[string]curve, error) {
	if len(cfg.Zones) == 0 {
		return nil, errors.New("curve: no zones")
	}
	if cfg.Open < 0 || cfg.Close > 24 || cfg.Open >= cfg.Close {
		return nil, fmt.Errorf("curve: business hours %d-%d", cfg.Open, cfg.Close)
	}
	zones := make([]*time.Location, 0, len(cfg.Zones))
	for _, z := range cfg.Zones {
		loc, err := time.LoadLocation(z)
		if err != nil {
			return nil, fmt.Errorf("curve: zone %q: %w", z, err)
		}
		zones = append(zones, loc)
	}
	return map[string]curve{
		"flat":           flatCurve{},
		"business_hours": newShaped(businessRaw(cfg, zones)),
		"nightly":        newShaped(nightlyRaw(zones[0])),
	}, nil
}

type flatCurve struct{}

func (flatCurve) factor(time.Time) float64 { return 1 }
func (flatCurve) max() float64             { return 1 }

type shapedCurve struct {
	raw  func(time.Time) float64
	norm float64
	peak float64
}

func (s *shapedCurve) factor(t time.Time) float64 { return s.raw(t) / s.norm }
func (s *shapedCurve) max() float64               { return s.peak }

// refWeek is the week the shapes are sampled over. January keeps US zones off daylight time; the shape moves with
// the clocks, so its average doesn't change.
var refWeek = time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC) // a Monday

// newShaped normalizes raw to a weekly mean of 1. raw changes only on whole local hours, so 5-minute samples are exact.
func newShaped(raw func(time.Time) float64) *shapedCurve {
	var sum, peak float64
	n := 0
	for t := refWeek; t.Before(refWeek.Add(7 * 24 * time.Hour)); t = t.Add(5 * time.Minute) {
		v := raw(t)
		sum += v
		n++
		if v > peak {
			peak = v
		}
	}
	mean := sum / float64(n)
	return &shapedCurve{raw: raw, norm: mean, peak: peak / mean}
}

func isWeekend(t time.Time) bool { return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday }

func businessRaw(cfg curveConfig, zones []*time.Location) func(time.Time) float64 {
	base := func(t time.Time) float64 {
		var s float64
		for _, z := range zones {
			lt := t.In(z)
			switch {
			case isWeekend(lt):
				s += cfg.Weekend
			case lt.Hour() >= cfg.Open && lt.Hour() < cfg.Close:
				s++
			default:
				s += cfg.OffHours
			}
		}
		return s / float64(len(zones))
	}
	// The busy hours are a multiple of the weekday's hourly average, before they're added.
	var sum float64
	n := 0
	for t := refWeek; t.Before(refWeek.Add(5 * 24 * time.Hour)); t = t.Add(5 * time.Minute) {
		sum += base(t)
		n++
	}
	busyLevel := cfg.BusyX * sum / float64(n)
	busy := make(map[int]struct{}, len(cfg.BusyHours))
	for _, h := range cfg.BusyHours {
		busy[h] = struct{}{}
	}
	return func(t time.Time) float64 {
		v := base(t)
		lt := t.In(zones[0])
		if _, ok := busy[lt.Hour()]; ok && !isWeekend(lt) && busyLevel > v {
			v = busyLevel
		}
		return v
	}
}

// nightlyRaw is for maintenance-style work: 01:00-05:00 local, every day.
func nightlyRaw(zone *time.Location) func(time.Time) float64 {
	return func(t time.Time) float64 {
		if h := t.In(zone).Hour(); h >= 1 && h < 5 {
			return 1
		}
		return 0
	}
}
