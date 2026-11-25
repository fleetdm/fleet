package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// spec is one workload action in the scenario catalogue's format (kind: action). Other kinds (storm, check, cycle)
// are skipped until the driver runs them.
type spec struct {
	Kind      string         `yaml:"kind"`
	ID        string         `yaml:"id"`
	Feature   string         `yaml:"feature"`
	Title     string         `yaml:"title"`
	Verb      string         `yaml:"verb"`
	With      map[string]any `yaml:"with"`
	Rate      rateSpec       `yaml:"rate"`
	Curve     string         `yaml:"curve"`
	Check     checkSpec      `yaml:"check"`
	Conflicts []string       `yaml:"conflicts"`
	Pool      *poolSpec      `yaml:"pool"`

	file     string
	rate     rate   // resolved: the profile's rate × growth × share, or the start guess
	rateFrom string // "profile" or "start"
	curve    curve
	budget   time.Duration // check.request.took
	verb     verb
}

type rateSpec struct {
	Start   string  `yaml:"start"`   // public start guess, used when the profile has no rate for the key
	Profile string  `yaml:"profile"` // key in the private profile's rates
	Share   float64 `yaml:"share"`   // this spec's share of the profile rate, when several specs split one key (default 1)
}

type checkSpec struct {
	Request struct {
		Status string `yaml:"status"`
		Took   string `yaml:"took"` // "<= 2s"
	} `yaml:"request"`
	Outcome map[string]string `yaml:"outcome"`
}

type poolSpec struct {
	Object string `yaml:"object"`
	Target int    `yaml:"target"`
	Band   int    `yaml:"band"`
}

// loadSpecs reads every YAML file under dir. It returns the action specs sorted by ID and the IDs of specs of other
// kinds, which the driver doesn't run yet.
func loadSpecs(dir string) ([]*spec, []string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			paths = append(paths, path)
		}
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	var specs []*spec
	var other []string
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		for {
			var s spec
			if err := dec.Decode(&s); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, nil, fmt.Errorf("%s: %w", path, err)
			}
			s.file = path
			if s.Kind != "action" {
				other = append(other, s.ID)
				continue
			}
			specs = append(specs, &s)
		}
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	seen := make(map[string]string, len(specs))
	for _, s := range specs {
		if s.ID == "" || s.Verb == "" {
			return nil, nil, fmt.Errorf("%s: id and verb are required", s.file)
		}
		if prev, ok := seen[s.ID]; ok {
			return nil, nil, fmt.Errorf("spec %s is in both %s and %s", s.ID, prev, s.file)
		}
		seen[s.ID] = s.file
	}
	return specs, other, nil
}

// resolve sets each spec's rate, curve, budget and verb. A missing profile rate falls back to the start guess.
func resolve(specs []*spec, prof *profile, cs map[string]curve) error {
	for _, s := range specs {
		share := s.Rate.Share
		if share == 0 {
			share = 1
		}
		if share < 0 || share > 1 {
			return fmt.Errorf("%s: rate.share %v is outside 0-1", s.ID, share)
		}
		r, ok, err := prof.rateFor(s.Rate.Profile)
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		switch {
		case ok:
			s.rate, s.rateFrom = rate(float64(r)*share), "profile"
		case s.Rate.Start != "":
			r, err := parseRate(s.Rate.Start)
			if err != nil {
				return fmt.Errorf("%s: %w", s.ID, err)
			}
			s.rate, s.rateFrom = r, "start"
		default:
			return fmt.Errorf("%s: no rate: the profile has no %q and rate.start is empty", s.ID, s.Rate.Profile)
		}

		name := s.Curve
		if name == "" {
			name = "flat"
		}
		c, ok := cs[name]
		if !ok {
			return fmt.Errorf("%s: unknown curve %q", s.ID, name)
		}
		s.curve = c

		if took := s.Check.Request.Took; took != "" {
			d, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(took), "<=")))
			if err != nil {
				return fmt.Errorf("%s: check.request.took %q: %w", s.ID, took, err)
			}
			s.budget = d
		}

		factory, ok := verbs[s.Verb]
		if !ok {
			return fmt.Errorf("%s: unknown verb %q (known: %s)", s.ID, s.Verb, strings.Join(verbNames(), ", "))
		}
		p := newParams(s.With)
		v, err := factory(p)
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		if err := p.unused(); err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		s.verb = v
	}
	return nil
}

func verbNames() []string {
	names := make([]string, 0, len(verbs))
	for n := range verbs {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// params reads a spec's `with` block and remembers which keys a verb used, so a typo is an error, not a default.
type params struct {
	m    map[string]any
	used map[string]bool
}

func newParams(m map[string]any) *params { return &params{m: m, used: map[string]bool{}} }

func (p *params) get(key string) (any, bool) {
	p.used[key] = true
	v, ok := p.m[key]
	return v, ok
}

func (p *params) unused() error {
	var extra []string
	for k := range p.m {
		if !p.used[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		slices.Sort(extra)
		return fmt.Errorf("with: unknown keys %s", strings.Join(extra, ", "))
	}
	return nil
}

func (p *params) str(key, def string) (string, error) {
	v, ok := p.get(key)
	if !ok {
		return def, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("with.%s: want a string, got %v", key, v)
	}
	return s, nil
}

func (p *params) integer(key string, def int) (int, error) {
	v, ok := p.get(key)
	if !ok {
		return def, nil
	}
	n, ok := v.(int)
	if !ok {
		return 0, fmt.Errorf("with.%s: want an integer, got %v", key, v)
	}
	return n, nil
}

func (p *params) float(key string, def float64) (float64, error) {
	v, ok := p.get(key)
	if !ok {
		return def, nil
	}
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case float64:
		return n, nil
	}
	return 0, fmt.Errorf("with.%s: want a number, got %v", key, v)
}

func (p *params) boolean(key string, def bool) (bool, error) {
	v, ok := p.get(key)
	if !ok {
		return def, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("with.%s: want true or false, got %v", key, v)
	}
	return b, nil
}

func (p *params) duration(key string, def time.Duration) (time.Duration, error) {
	s, err := p.str(key, "")
	if err != nil || s == "" {
		return def, err
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("with.%s: %w", key, err)
	}
	return d, nil
}

// intRange reads "1-50" or a single number.
func (p *params) intRange(key string, defMin, defMax int) (int, int, error) {
	v, ok := p.get(key)
	if !ok {
		return defMin, defMax, nil
	}
	if n, ok := v.(int); ok {
		return n, n, nil
	}
	s, ok := v.(string)
	if !ok {
		return 0, 0, fmt.Errorf("with.%s: want N or N-M, got %v", key, v)
	}
	var lo, hi int
	if _, err := fmt.Sscanf(s, "%d-%d", &lo, &hi); err != nil || lo < 1 || hi < lo {
		return 0, 0, fmt.Errorf("with.%s: want N-M with 1 <= N <= M, got %q", key, s)
	}
	return lo, hi, nil
}
