package main

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// profile is the private part of the workload: a customer's measured activity per day and the growth to apply on
// top. It lives outside the repository (a local file, or an object the ECS task reads), and the driver never logs
// its numbers.
//
//	growth: 1.3
//	rates:
//	  activity.live_query: 100/day
//	  activity.user_logged_in: 50/day
//	curve:              # optional, overrides defaultCurveConfig field by field
//	  busy_hours: [10, 14]
type profile struct {
	Growth float64           `yaml:"growth"`
	Rates  map[string]string `yaml:"rates"`
	Curve  *curveConfig      `yaml:"curve"`
}

func loadProfile(path string) (*profile, error) {
	if path == "" {
		return &profile{Growth: 1}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var p profile
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("profile %s: %w", path, err)
	}
	if p.Growth == 0 {
		p.Growth = 1
	}
	if p.Growth < 0 {
		return nil, fmt.Errorf("profile %s: growth %v", path, p.Growth)
	}
	for k, v := range p.Rates {
		if _, err := parseRate(v); err != nil {
			return nil, fmt.Errorf("profile %s: rates.%s: %w", path, k, err)
		}
	}
	return &p, nil
}

// rateFor returns the profile's rate for key, with growth applied.
func (p *profile) rateFor(key string) (rate, bool, error) {
	s, ok := p.Rates[key]
	if key == "" || !ok {
		return 0, false, nil
	}
	r, err := parseRate(s)
	if err != nil {
		return 0, false, err
	}
	return rate(float64(r) * p.Growth), true, nil
}

// curveConfig returns the defaults with the profile's overrides applied.
func (p *profile) curveConfig() curveConfig {
	c := defaultCurveConfig()
	o := p.Curve
	if o == nil {
		return c
	}
	if len(o.Zones) > 0 {
		c.Zones = o.Zones
	}
	if o.Open != 0 || o.Close != 0 {
		c.Open, c.Close = o.Open, o.Close
	}
	if o.OffHours != 0 {
		c.OffHours = o.OffHours
	}
	if o.Weekend != 0 {
		c.Weekend = o.Weekend
	}
	if o.BusyHours != nil {
		c.BusyHours = o.BusyHours
	}
	if o.BusyX != 0 {
		c.BusyX = o.BusyX
	}
	return c
}
