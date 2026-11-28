package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseRate(t *testing.T) {
	for in, perDay := range map[string]float64{"40/day": 40, "3/h": 72, "0.5/min": 720, "7/week": 1, "0/day": 0} {
		r, err := parseRate(in)
		require.NoError(t, err, in)
		assert.InDelta(t, perDay, r.perDay(), 1e-9, in)
	}
	for _, in := range []string{"40", "x/day", "-1/day", "3/fortnight"} {
		_, err := parseRate(in)
		assert.Error(t, err, in)
	}
}

func TestCurvesAverageOne(t *testing.T) {
	cs, err := curves(defaultCurveConfig())
	require.NoError(t, err)
	for name, c := range cs {
		var sum float64
		n := 0
		peak := 0.0
		// A week other than the reference one, to check the shape doesn't depend on it.
		start := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC) // no DST switch: the nightly curve loses an hour on that night
		for ts := start; ts.Before(start.Add(7 * 24 * time.Hour)); ts = ts.Add(time.Minute) {
			f := c.factor(ts)
			sum += f
			n++
			peak = math.Max(peak, f)
		}
		assert.InDelta(t, 1, sum/float64(n), 0.03, name)
		assert.LessOrEqual(t, peak, c.max()+1e-9, name)
	}
}

func TestBusyHours(t *testing.T) {
	cfg := defaultCurveConfig()
	cs, err := curves(cfg)
	require.NoError(t, err)
	c, ok := cs["business_hours"]
	require.True(t, ok)
	ny, err := time.LoadLocation(cfg.Zones[0])
	require.NoError(t, err)
	wed := func(h int) time.Time { return time.Date(2026, 1, 7, h, 30, 0, 0, ny) }
	assert.Greater(t, c.factor(wed(10)), c.factor(wed(12)), "a busy hour runs hotter than a normal business hour")
	assert.Greater(t, c.factor(wed(12)), c.factor(wed(3)), "business hours run hotter than the night")
	sat := time.Date(2026, 1, 10, 10, 30, 0, 0, ny)
	assert.Less(t, c.factor(sat), c.factor(wed(12)), "weekends are quiet")
}

func TestNextArrivalMatchesExpected(t *testing.T) {
	cs, err := curves(defaultCurveConfig())
	require.NoError(t, err)
	r, err := parseRate("400/day")
	require.NoError(t, err)
	rng := newSeeded(1)
	start := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	end := start.Add(28 * 24 * time.Hour)
	for _, name := range []string{"flat", "business_hours"} {
		c, ok := cs[name]
		require.True(t, ok)
		n := 0
		for ts := nextArrival(rng, start, r, c); ts.Before(end); ts = nextArrival(rng, ts, r, c) {
			n++
		}
		want := expected(r, c, start, end) // ~11,200; Poisson sd ~106
		assert.InDelta(t, want, float64(n), 4*math.Sqrt(want), name)
	}
	zero, err := parseRate("0/day")
	require.NoError(t, err)
	assert.True(t, nextArrival(rng, start, zero, flatCurve{}).After(end))
}

func TestLoadAndResolveScenarios(t *testing.T) {
	specs, checks, other, err := loadSpecs("../scenarios")
	require.NoError(t, err)
	assert.Empty(t, other)
	require.NotEmpty(t, specs)
	cs, err := curves(defaultCurveConfig())
	require.NoError(t, err)
	prof, err := loadProfile("")
	require.NoError(t, err)
	require.NoError(t, resolve(specs, prof, cs))
	for _, s := range specs {
		assert.Equal(t, "start", s.rateFrom, s.ID)
		assert.Positive(t, float64(s.rate), s.ID)
		assert.NotNil(t, s.verb, s.ID)
	}

	require.NotEmpty(t, checks)
	mcs, dropped, err := resolveChecks(checks, &metricSources{guard: &guard{}, aws: &awsSources{lt: "fleet-test"}})
	require.NoError(t, err)
	assert.Empty(t, dropped)
	assert.NotEmpty(t, mcs)
	// Without AWS, only the driver's own metrics stay.
	mcs, dropped, err = resolveChecks(checks, &metricSources{guard: &guard{}})
	require.NoError(t, err)
	assert.NotEmpty(t, dropped)
	for _, mc := range mcs {
		assert.Contains(t, []string{"healthz.p99", "hosts.online_drop"}, mc.metric)
	}
}

func TestBudgets(t *testing.T) {
	for _, c := range []struct {
		in     string
		unit   unit
		v      float64
		within bool
	}{
		{"<= 2s", unitSeconds, 2, true},
		{"<= 2s", unitSeconds, 2.01, false},
		{"< 500ms", unitSeconds, 0.4, true},
		{"< 60%", unitPercent, 60, false},
		{"== 0", unitCount, 0, true},
		{"== 0", unitCount, 1, false},
		{">= 10", unitCount, 12, true},
	} {
		b, err := parseBudget(c.in)
		require.NoError(t, err, c.in)
		assert.Equal(t, c.unit, b.unit, c.in)
		assert.Equal(t, c.within, b.within(c.v), "%s with %v", c.in, c.v)
	}
	for _, bad := range []string{"", "2s", "<= trailing_7d_p50 × 1.5", "< 2 parsecs"} {
		_, err := parseBudget(bad)
		require.Error(t, err, bad)
	}

	n, err := parseBreach("3 windows in a row")
	require.NoError(t, err)
	assert.Equal(t, 3, n)
}

type fakeSource struct{ vals []float64 }

func (s *fakeSource) lag() time.Duration { return 0 }

func (s *fakeSource) read(context.Context, time.Time, time.Time) (reading, error) {
	v := s.vals[0]
	s.vals = s.vals[1:]
	return reading{value: v}, nil
}

func TestCheckBreachTrips(t *testing.T) {
	var buf bytes.Buffer
	events := newEventLog("test", &buf)
	stop := filepath.Join(t.TempDir(), "STOP")
	g := &guard{stop: &fileMarker{path: stop}, events: events, logger: slog.New(slog.DiscardHandler), api: &api{}}
	b, err := parseBudget("< 60%")
	require.NoError(t, err)
	mc := &metricCheck{checkID: "C-X", metric: "fleet.memory.max", window: 5 * time.Minute, budget: b, breachN: 2, trip: true,
		src: &fakeSource{vals: []float64{70, 50, 70, 70}}}
	r := &checkRunner{checks: []*metricCheck{mc}, guard: g, events: events, logger: g.logger}
	now := time.Now()
	r.evaluate(t.Context(), mc, now) // breach 1
	r.evaluate(t.Context(), mc, now) // ok: the streak resets
	r.evaluate(t.Context(), mc, now) // breach 1
	assert.NoFileExists(t, stop)
	r.evaluate(t.Context(), mc, now) // breach 2: held, trips
	assert.FileExists(t, stop)

	var verdicts []string
	for line := range bytes.SplitSeq(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var e event
		require.NoError(t, json.Unmarshal(line, &e))
		if e.Kind == "check" {
			verdicts = append(verdicts, e.Verdict)
		}
	}
	assert.Equal(t, []string{"breach", "ok", "breach", "breach"}, verdicts)
}

func TestProfileRates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte("growth: 1.5\nrates:\n  activity.live_query: 100/day\ncurve:\n  busy_x: 2\n"), 0o600))
	prof, err := loadProfile(path)
	require.NoError(t, err)
	r, ok, err := prof.rateFor("activity.live_query")
	require.NoError(t, err)
	require.True(t, ok)
	assert.InDelta(t, 150, r.perDay(), 1e-9)
	_, ok, err = prof.rateFor("activity.unknown")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.InDelta(t, 2.0, prof.curveConfig().BusyX, 1e-9)
	assert.Equal(t, defaultCurveConfig().Zones, prof.curveConfig().Zones)

	require.NoError(t, os.WriteFile(path, []byte("rates:\n  a: 1/day\nbogus: 1\n"), 0o600))
	_, err = loadProfile(path)
	assert.Error(t, err, "unknown keys are rejected")
}

func TestResolveErrors(t *testing.T) {
	cs, err := curves(defaultCurveConfig())
	require.NoError(t, err)
	prof, err := loadProfile("")
	require.NoError(t, err)
	base := func() *spec {
		return &spec{ID: "X", Kind: "action", Verb: "hosts.idp", Rate: rateSpec{Start: "1/day"}, Curve: "flat"}
	}
	cases := map[string]func(s *spec){
		"unknown verb":     func(s *spec) { s.Verb = "nope" },
		"unknown with key": func(s *spec) { s.With = map[string]any{"userz": 3} },
		"bad with type":    func(s *spec) { s.With = map[string]any{"users": "many"} },
		"unknown curve":    func(s *spec) { s.Curve = "weekends" },
		"no rate":          func(s *spec) { s.Rate = rateSpec{Profile: "activity.x"} },
		"bad share":        func(s *spec) { s.Rate.Share = 2 },
		"bad budget":       func(s *spec) { s.Check.Request.Took = "<= soon" },
	}
	for name, mod := range cases {
		s := base()
		mod(s)
		require.Error(t, resolve([]*spec{s}, prof, cs), name)
	}
	s := base()
	require.NoError(t, resolve([]*spec{s}, prof, cs))
}

func TestIntRange(t *testing.T) {
	p := newParams(map[string]any{"a": "1-50", "b": 7, "c": "50-1"})
	lo, hi, err := p.intRange("a", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, [2]int{1, 50}, [2]int{lo, hi})
	lo, hi, err = p.intRange("b", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, [2]int{7, 7}, [2]int{lo, hi})
	_, _, err = p.intRange("c", 0, 0)
	require.Error(t, err)
	lo, hi, err = p.intRange("missing", 2, 3)
	require.NoError(t, err)
	assert.Equal(t, [2]int{2, 3}, [2]int{lo, hi})
}

func TestGitopsEdits(t *testing.T) {
	repo := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(repo, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(s), 0o644))
	}
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		require.NoError(t, err)
		return string(b)
	}
	write("lib/a/scripts/one.sh", "#!/bin/sh\necho hi")
	write("lib/a/apple/p.mobileconfig", "<?xml version=\"1.0\"?>\n<plist version=\"1.0\">\n<dict/>\n</plist>\n")
	write("fleets/a.yml", "name: A\nagent_options:\n  config:\n    options:\n      distributed_tls_max_attempts: 3\n")

	v, err := newGitopsApply(newParams(map[string]any{"repo": repo, "command": "true", "scripts": 1, "profiles": 1, "agent_options": 1}))
	require.NoError(t, err)
	g := v.(*gitopsApply)
	edits, err := g.edit("2026-01-01T00:00:00Z", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"scripts": 1, "profiles": 1, "agent_options": 1}, edits)
	assert.Equal(t, "#!/bin/sh\necho hi\n# fleet-driver edit 2026-01-01T00:00:00Z\n", read("lib/a/scripts/one.sh"))
	assert.Contains(t, read("lib/a/apple/p.mobileconfig"), "<plist version=\"1.0\">\n<!-- fleet-driver edit 2026-01-01T00:00:00Z -->\n<dict/>")
	assert.Contains(t, read("fleets/a.yml"), "distributed_tls_max_attempts: 4\n")

	_, err = g.edit("2026-01-02T00:00:00Z", nil)
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/sh\necho hi\n# fleet-driver edit 2026-01-02T00:00:00Z\n", read("lib/a/scripts/one.sh"), "the marker is replaced, not added")
	assert.Equal(t, 1, strings.Count(read("lib/a/apple/p.mobileconfig"), "fleet-driver edit"))
	assert.Contains(t, read("fleets/a.yml"), "distributed_tls_max_attempts: 3\n", "the option alternates")

	require.NoError(t, v.setup(t.Context(), &env{}))
	r := v.run(t.Context(), &env{stateDir: t.TempDir()})
	require.NoError(t, r.err)
	assert.Equal(t, 200, r.status)
}

func TestGitopsLabels(t *testing.T) {
	repo := t.TempDir()
	hosts := &hostCache{}
	for i := range 50 {
		hosts.hosts = append(hosts.hosts, hostInfo{ID: uint(i + 1), UUID: fmt.Sprintf("uuid-%02d", i)}) //nolint:gosec // small test numbers
	}
	v, err := newGitopsApply(newParams(map[string]any{"repo": repo, "command": "true", "scripts": 0, "labels": 3, "label_hosts": 10, "label_change": 2}))
	require.NoError(t, err)
	g := v.(*gitopsApply)

	edits, err := g.edit("t1", hosts)
	require.NoError(t, err)
	assert.Equal(t, 3, edits["labels"])
	read := func() []gitopsLabel {
		b, err := os.ReadFile(filepath.Join(repo, "labels/driver.yml"))
		require.NoError(t, err)
		var ls []gitopsLabel
		require.NoError(t, yaml.Unmarshal(b, &ls))
		return ls
	}
	first := read()
	require.Len(t, first, 3)
	assert.Equal(t, "driver-gitops-label-01", first[0].Name)
	assert.Equal(t, "manual", first[0].LabelMembershipType)
	assert.Len(t, first[0].Hosts, 10)

	_, err = g.edit("t2", hosts)
	require.NoError(t, err)
	second := read()
	for i := range second {
		assert.Len(t, second[i].Hosts, 10, "two out, two in")
		assert.NotEqual(t, first[i].Hosts, second[i].Hosts)
	}
}

type fixedVerb struct {
	sleep time.Duration
	r     result
}

func (v fixedVerb) setup(context.Context, *env) error { return nil }

func (v fixedVerb) run(context.Context, *env) result {
	time.Sleep(v.sleep)
	return v.r
}

func TestRunOneBudgetUsesActionTime(t *testing.T) {
	var buf bytes.Buffer
	events := newEventLog("test", &buf)
	s := &spec{ID: "X", Verb: "x", budget: 50 * time.Millisecond}

	// Slow lookups, fast action: ok, and took_ms is the action's.
	s.verb = fixedVerb{sleep: 80 * time.Millisecond, r: result{status: 200, took: 10 * time.Millisecond}}
	runOne(t.Context(), s, &env{}, events)
	// No action time reported: the whole run counts.
	s.verb = fixedVerb{sleep: 80 * time.Millisecond, r: result{status: 200}}
	runOne(t.Context(), s, &env{}, events)

	var evs []event
	for line := range bytes.SplitSeq(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var e event
		require.NoError(t, json.Unmarshal(line, &e))
		evs = append(evs, e)
	}
	require.Len(t, evs, 2)
	assert.Equal(t, "ok", evs[0].Verdict)
	assert.EqualValues(t, 10, evs[0].TookMS)
	assert.GreaterOrEqual(t, evs[0].Detail["total_ms"], float64(80))
	assert.Equal(t, "slow", evs[1].Verdict)
	assert.GreaterOrEqual(t, evs[1].TookMS, int64(80))
}

func TestDerivePassword(t *testing.T) {
	a := derivePassword("0123456789abcdef", "driver-user-001@example.com")
	assert.Equal(t, a, derivePassword("0123456789abcdef", "driver-user-001@example.com"))
	assert.NotEqual(t, a, derivePassword("0123456789abcdef", "driver-user-002@example.com"))
	assert.NotEqual(t, a, derivePassword("0123456789abcdeX", "driver-user-001@example.com"))
	assert.GreaterOrEqual(t, len(a), 12)
	assert.LessOrEqual(t, len(a), 48)
}

func TestFileMarker(t *testing.T) {
	m := &fileMarker{path: filepath.Join(t.TempDir(), "STOP")}
	_, ok, err := m.get(t.Context())
	require.NoError(t, err)
	assert.False(t, ok)
	require.NoError(t, m.create(t.Context(), "first"))
	require.NoError(t, m.create(t.Context(), "second")) // an existing marker keeps its reason
	line, ok, err := m.get(t.Context())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "first", line)
}
