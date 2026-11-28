package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// checkSpec is a kind: check spec: one metric watched all the time (FORMAT.md's single-metric shape), or several
// under one ID with `metrics`. Each metric is read every `every` over the last `window`; `breach` windows in a row
// over budget hold a breach, which trips the workload when `trip` is set and is a warning otherwise. Without a
// budget the metric is only recorded.
type checkSpec struct {
	Kind    string       `yaml:"kind"`
	ID      string       `yaml:"id"`
	Title   string       `yaml:"title"`
	Covers  []int        `yaml:"covers"`
	Metrics []metricSpec `yaml:"metrics"`
	// The invariant shape (every + compare) isn't run yet; a spec with compare is skipped.
	Compare   []string `yaml:"compare"`
	Tolerance string   `yaml:"tolerance"`

	metricSpec `yaml:",inline"`

	file string
}

type metricSpec struct {
	Metric string `yaml:"metric"`
	Window string `yaml:"window"`
	Every  string `yaml:"every"`
	Budget string `yaml:"budget"`
	Breach any    `yaml:"breach"` // N, or "N windows in a row"
	Trip   bool   `yaml:"trip"`
}

// metricCheck is one metric of a check, resolved.
type metricCheck struct {
	checkID string
	metric  string
	window  time.Duration
	every   time.Duration
	budget  *budget
	breachN int
	trip    bool
	src     source

	streak int
}

// budget is "<op> <value><unit>": "<= 2s", "< 60%", "== 0".
type budget struct {
	raw   string
	op    string
	value float64 // seconds for durations
	unit  unit
}

type unit int

const (
	unitCount unit = iota
	unitSeconds
	unitPercent
)

func (u unit) String() string {
	return [...]string{"count", "seconds", "percent"}[u]
}

func parseBudget(s string) (*budget, error) {
	s = strings.TrimSpace(s)
	b := &budget{raw: s}
	for _, op := range []string{"<=", ">=", "==", "<", ">"} {
		if rest, ok := strings.CutPrefix(s, op); ok {
			b.op, s = op, strings.TrimSpace(rest)
			break
		}
	}
	if b.op == "" {
		return nil, fmt.Errorf("budget %q: want <op> <value>, e.g. \"<= 2s\"", b.raw)
	}
	switch {
	case strings.HasSuffix(s, "%"):
		b.unit = unitPercent
		s = strings.TrimSuffix(s, "%")
	case strings.HasSuffix(s, "s"):
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("budget %q: %w", b.raw, err)
		}
		b.unit, b.value = unitSeconds, d.Seconds()
		return b, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("budget %q: %q isn't a number (relative budgets like trailing baselines aren't supported yet)", b.raw, s)
	}
	b.value = v
	return b, nil
}

// within reports whether v meets the budget.
func (b *budget) within(v float64) bool {
	switch b.op {
	case "<=":
		return v <= b.value
	case ">=":
		return v >= b.value
	case "==":
		return v == b.value
	case "<":
		return v < b.value
	case ">":
		return v > b.value
	}
	return false
}

func parseBreach(v any) (int, error) {
	switch b := v.(type) {
	case nil:
		return 1, nil
	case int:
		if b < 1 {
			return 0, fmt.Errorf("breach %d: want 1 or more", b)
		}
		return b, nil
	case string:
		var n int
		if _, err := fmt.Sscanf(b, "%d", &n); err != nil || n < 1 {
			return 0, fmt.Errorf("breach %q: want N or \"N windows in a row\"", b)
		}
		return n, nil
	}
	return 0, fmt.Errorf("breach %v: want N or \"N windows in a row\"", v)
}

// resolveChecks turns check specs into metric checks. A metric whose source isn't available here (no AWS
// configuration, say) is dropped with a reason rather than failing the run.
func resolveChecks(specs []*checkSpec, ms *metricSources) ([]*metricCheck, []string, error) {
	var out []*metricCheck
	var dropped []string
	for _, c := range specs {
		if len(c.Compare) > 0 {
			dropped = append(dropped, c.ID+": compare checks aren't run yet")
			continue
		}
		list := c.Metrics
		if c.Metric != "" {
			if len(list) > 0 {
				return nil, nil, fmt.Errorf("%s: set metric or metrics, not both", c.ID)
			}
			list = []metricSpec{c.metricSpec}
		}
		if len(list) == 0 {
			return nil, nil, fmt.Errorf("%s: no metric", c.ID)
		}
		for _, m := range list {
			mc, err := resolveMetric(c.ID, m)
			if err != nil {
				return nil, nil, fmt.Errorf("%s %s: %w", c.ID, m.Metric, err)
			}
			src, u, err := ms.source(m.Metric)
			if err != nil {
				if errors.Is(err, errSourceUnavailable) {
					dropped = append(dropped, fmt.Sprintf("%s %s: %v", c.ID, m.Metric, err))
					continue
				}
				return nil, nil, fmt.Errorf("%s: %w", c.ID, err)
			}
			if mc.budget != nil && mc.budget.unit != u {
				return nil, nil, fmt.Errorf("%s %s: budget %q is in %s, the metric in %s", c.ID, m.Metric, mc.budget.raw, mc.budget.unit, u)
			}
			mc.src = src
			out = append(out, mc)
		}
	}
	return out, dropped, nil
}

func resolveMetric(id string, m metricSpec) (*metricCheck, error) {
	mc := &metricCheck{checkID: id, metric: m.Metric, trip: m.Trip}
	var err error
	if mc.window, err = time.ParseDuration(m.Window); err != nil || mc.window < time.Minute {
		return nil, fmt.Errorf("window %q: want a duration of 1m or more", m.Window)
	}
	mc.every = mc.window
	if m.Every != "" {
		if mc.every, err = time.ParseDuration(m.Every); err != nil || mc.every < time.Minute {
			return nil, fmt.Errorf("every %q: want a duration of 1m or more", m.Every)
		}
	}
	if strings.TrimSpace(m.Budget) != "" { // without one, the metric is only recorded
		if mc.budget, err = parseBudget(m.Budget); err != nil {
			return nil, err
		}
	}
	if mc.breachN, err = parseBreach(m.Breach); err != nil {
		return nil, err
	}
	if mc.trip && mc.budget == nil {
		return nil, errors.New("trip needs a budget")
	}
	return mc, nil
}

// checkRunner reads every metric check on its own schedule until ctx ends. Checks keep running while the workload
// is paused, so recovery is on record too.
type checkRunner struct {
	checks []*metricCheck
	guard  *guard
	events *eventLog
	logger *slog.Logger
}

func (r *checkRunner) run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, mc := range r.checks {
		wg.Go(func() {
			t := time.NewTicker(mc.every)
			defer t.Stop()
			for {
				r.evaluate(ctx, mc, time.Now())
				select {
				case <-ctx.Done():
					return
				case <-t.C:
				}
			}
		})
	}
	wg.Wait()
}

// evaluate reads one window ending at now minus the source's lag and writes its event.
func (r *checkRunner) evaluate(ctx context.Context, mc *metricCheck, now time.Time) {
	to := now.Add(-mc.src.lag()).Truncate(time.Second)
	from := to.Add(-mc.window)
	rd, err := mc.src.read(ctx, from, to)
	if ctx.Err() != nil {
		return
	}
	detail := map[string]any{"metric": mc.metric}
	maps.Copy(detail, rd.detail)
	ev := event{Kind: "check", ID: mc.checkID, Start: ts(from), End: ts(to), Detail: detail}
	switch {
	case err != nil:
		ev.Verdict, ev.Error = "unreadable", err.Error()
	case rd.none:
		ev.Verdict = "no_data"
	default:
		detail["value"] = round3(rd.value)
		if mc.budget == nil {
			ev.Verdict = "recorded"
			break
		}
		detail["budget"] = mc.budget.raw
		if mc.budget.within(rd.value) {
			ev.Verdict, mc.streak = "ok", 0
			break
		}
		mc.streak++
		ev.Verdict = "breach"
		detail["streak"], detail["breach_n"] = mc.streak, mc.breachN
		if mc.streak >= mc.breachN {
			detail["held"] = true
		}
	}
	r.events.write(ev)

	if ev.Verdict != "breach" || mc.streak != mc.breachN {
		return
	}
	why := fmt.Sprintf("%s %s = %v, budget %s, %d window(s) of %s in a row", mc.checkID, mc.metric, round3(rd.value),
		mc.budget.raw, mc.streak, mc.window)
	if mc.trip {
		r.guard.trip(ctx, why)
		return
	}
	r.logger.WarnContext(ctx, "check breach", "check", mc.checkID, "metric", mc.metric, "value", round3(rd.value), "budget", mc.budget.raw)
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
