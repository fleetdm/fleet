package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

// never is the arrival time of an action with a zero rate.
const never = 100 * 365 * 24 * time.Hour

// nextArrival draws the next arrival after from, for a Poisson process with rate r × c.factor(t): candidates at the
// curve's peak rate, each kept with probability factor/max (thinning).
func nextArrival(rng *rand.Rand, from time.Time, r rate, c curve) time.Time {
	peak := float64(r) * c.max()
	if peak <= 0 {
		return from.Add(never)
	}
	t := from
	for range 1_000_000 {
		t = t.Add(time.Duration(rng.ExpFloat64() / peak * float64(time.Second)))
		if rng.Float64()*c.max() <= c.factor(t) {
			return t
		}
	}
	return from.Add(never)
}

// expected integrates rate × factor over [from, to) in one-minute steps.
func expected(r rate, c curve, from, to time.Time) float64 {
	var n float64
	for t := from; t.Before(to); t = t.Add(time.Minute) {
		n += float64(r) * c.factor(t) * 60
	}
	return n
}

// dryRun prints what the schedule would do over window, without calling Fleet.
func dryRun(w io.Writer, specs []*spec, rng *rand.Rand, start time.Time, window time.Duration) {
	start = start.UTC().Truncate(time.Hour)
	end := start.Add(window)
	hours := int(window / time.Hour)
	counts := make([][]int, len(specs))
	for i, s := range specs {
		counts[i] = make([]int, hours+1)
		for t := nextArrival(rng, start, s.rate, s.curve); t.Before(end); t = nextArrival(rng, t, s.rate, s.curve) {
			counts[i][int(t.Sub(start)/time.Hour)]++
		}
	}

	fmt.Fprintf(w, "Schedule from %s for %s (UTC). Rates come from the profile when it has them; don't paste this output publicly.\n\n",
		start.Format("2006-01-02 15:04"), window)
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVERB\tRATE\tFROM\tCURVE\tEXPECTED\tSCHEDULED")
	for i, s := range specs {
		total := 0
		for _, n := range counts[i] {
			total += n
		}
		curveName := s.Curve
		if curveName == "" {
			curveName = "flat"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%.0f\t%d\n", s.ID, s.Verb, s.rate, s.rateFrom, curveName,
			expected(s.rate, s.curve, start, end), total)
	}
	_ = tw.Flush()

	fmt.Fprintln(w)
	tw = tabwriter.NewWriter(w, 2, 4, 2, ' ', tabwriter.AlignRight)
	head := []string{"HOUR"}
	for _, s := range specs {
		head = append(head, s.ID)
	}
	fmt.Fprintln(tw, strings.Join(head, "\t")+"\t")
	for h := range hours {
		row := []string{start.Add(time.Duration(h) * time.Hour).Format("Mon 15h")}
		for i := range specs {
			row = append(row, fmt.Sprint(counts[i][h]))
		}
		fmt.Fprintln(tw, strings.Join(row, "\t")+"\t")
	}
	_ = tw.Flush()
}

// workload runs the specs until ctx ends: one arrival stream per spec, at most one run of a spec at a time, at most
// maxInflight runs in all. An arrival that can't run (paused, still running, too many in flight) is logged as
// skipped and not made up later, so a pause never turns into a burst.
type workload struct {
	specs       []*spec
	env         *env
	guard       *guard
	events      *eventLog
	rng         *rand.Rand
	maxInflight int
}

func (wl *workload) run(ctx context.Context) {
	sem := make(chan struct{}, wl.maxInflight)
	busy := make([]atomic.Bool, len(wl.specs))
	next := make([]time.Time, len(wl.specs))
	now := time.Now()
	for i, s := range wl.specs {
		next[i] = nextArrival(wl.rng, now, s.rate, s.curve)
	}
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		i := 0
		for j := range next {
			if next[j].Before(next[i]) {
				i = j
			}
		}
		timer := time.NewTimer(time.Until(next[i]))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		s := wl.specs[i]
		next[i] = nextArrival(wl.rng, next[i], s.rate, s.curve)

		skip := ""
		if p, why := wl.guard.paused(); p {
			skip = "paused: " + why
		} else if busy[i].Load() {
			skip = "previous run still going"
		}
		if skip == "" {
			select {
			case sem <- struct{}{}:
			default:
				skip = fmt.Sprintf("%d actions already in flight", wl.maxInflight)
			}
		}
		if skip != "" {
			wl.events.write(event{Kind: "action", ID: s.ID, Verb: s.Verb, Verdict: "skipped", Error: skip})
			continue
		}
		busy[i].Store(true)
		wg.Add(1)
		go func() {
			defer func() { busy[i].Store(false); <-sem; wg.Done() }()
			wl.guard.record(ctx, runOne(ctx, s, wl.env, wl.events))
		}()
	}
}

// runOne runs one action and writes its event.
func runOne(ctx context.Context, s *spec, e *env, events *eventLog) result {
	start := time.Now()
	r := s.verb.run(ctx, e)
	end := time.Now()
	took := end.Sub(start)
	if r.took > 0 {
		took = r.took
		if r.detail == nil {
			r.detail = map[string]any{}
		}
		r.detail["total_ms"] = end.Sub(start).Milliseconds()
	}
	ev := event{
		Kind: "action", ID: s.ID, Verb: s.Verb, Start: ts(start), End: ts(end),
		Status: r.status, TookMS: took.Milliseconds(), Detail: r.detail,
	}
	switch {
	case r.skip != "":
		ev.Verdict, ev.Error = "skipped", r.skip
	case r.err != nil && ctx.Err() != nil: // cut short by the driver stopping, not by Fleet
		ev.Verdict, ev.Error = "skipped", "driver stopped: "+r.err.Error()
	case r.err != nil:
		ev.Verdict, ev.Error = "broken", r.err.Error()
	case r.slow || (s.budget > 0 && took > s.budget):
		ev.Verdict = "slow"
	default:
		ev.Verdict = "ok"
	}
	events.write(ev)
	return r
}
