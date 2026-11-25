package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
)

// guard decides whether the workload runs. The stop file pauses it (whoever wrote it: this driver, a watcher, a
// person), and the driver's own wires write that file on a trip. Recovery is by hand: the workload resumes when
// someone removes the file. Only Fleet's own symptoms trip; database load alone doesn't (it's watched elsewhere and
// at this population it's high through every vulnerabilities run).
type guard struct {
	stopFile string
	onTrip   string
	api      *api
	events   *eventLog
	logger   *slog.Logger

	mu       sync.Mutex
	tripped  string
	isPaused bool
	recent   []bool // the last outcomes, true = broken on the server's side
	hzFails  int
	hz       []healthzReading // the last two hours, for the healthz.p99 check
}

type healthzReading struct {
	at   time.Time
	took time.Duration
	ok   bool
}

// healthzReadings returns the readings taken in [from, to).
func (g *guard) healthzReadings(from, to time.Time) []healthzReading {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []healthzReading
	for _, h := range g.hz {
		if !h.at.Before(from) && h.at.Before(to) {
			out = append(out, h)
		}
	}
	return out
}

const (
	brokenWindow = 20
	brokenTrip   = 10
	healthzEvery = time.Minute
	healthzSlow  = 2 * time.Second
	healthzTrip  = 3
)

// paused reports whether actions should be skipped now, and why. It writes pause and resume events on changes.
func (g *guard) paused() (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	reason := ""
	if g.stopFile != "" {
		if b, err := os.ReadFile(g.stopFile); err == nil {
			reason = "stop file: " + strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
		} else {
			g.tripped = "" // the file was removed: recovered by hand
		}
	}
	if reason == "" && g.tripped != "" {
		reason = g.tripped
	}
	switch {
	case reason != "" && !g.isPaused:
		g.isPaused = true
		g.events.write(event{Kind: "pause", Error: reason})
	case reason == "" && g.isPaused:
		g.isPaused, g.recent, g.hzFails = false, nil, 0
		g.events.write(event{Kind: "resume"})
	}
	return reason != "", reason
}

// record counts an action's outcome toward the broken-actions wire.
func (g *guard) record(ctx context.Context, r result) {
	broken := r.err != nil && (r.status == 0 || r.status >= 500) && ctx.Err() == nil
	g.mu.Lock()
	g.recent = append(g.recent, broken)
	if len(g.recent) > brokenWindow {
		g.recent = g.recent[1:]
	}
	n := 0
	for _, b := range g.recent {
		if b {
			n++
		}
	}
	g.mu.Unlock()
	if n >= brokenTrip {
		g.trip(ctx, fmt.Sprintf("%d of the last %d actions got a 5xx or no response", n, len(g.recent)))
	}
}

// watchHealthz trips after healthzTrip readings in a row that fail or take healthzSlow or longer.
func (g *guard) watchHealthz(ctx context.Context) {
	t := time.NewTicker(healthzEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		hctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		start := time.Now()
		status, err := g.api.doAs(hctx, "", http.MethodGet, "/healthz", nil, nil, nil)
		took := time.Since(start)
		cancel()
		if ctx.Err() != nil {
			return
		}
		g.mu.Lock()
		g.hz = append(g.hz, healthzReading{at: start, took: took, ok: err == nil && status == http.StatusOK})
		g.hz = slices.DeleteFunc(g.hz, func(h healthzReading) bool { return time.Since(h.at) > 2*time.Hour })
		if err != nil || status != http.StatusOK || took >= healthzSlow {
			g.hzFails++
		} else {
			g.hzFails = 0
		}
		fails := g.hzFails
		g.mu.Unlock()
		if fails >= healthzTrip {
			g.trip(ctx, fmt.Sprintf("healthz failing or >= %s %d times in a row (last: %d in %s)", healthzSlow, fails, status, took.Round(time.Millisecond)))
		}
	}
}

func (g *guard) trip(ctx context.Context, reason string) {
	g.mu.Lock()
	if g.tripped != "" {
		g.mu.Unlock()
		return
	}
	g.tripped = reason
	g.mu.Unlock()

	g.events.write(event{Kind: "trip", Error: reason})
	g.logger.ErrorContext(ctx, "trip: workload paused until the stop file is removed", "reason", reason, "stop_file", g.stopFile)
	if g.stopFile != "" {
		if _, err := os.Stat(g.stopFile); errors.Is(err, fs.ErrNotExist) {
			line := fmt.Sprintf("%s driver: %s\n", time.Now().UTC().Format("2006-01-02T15:04Z"), reason)
			if err := os.WriteFile(g.stopFile, []byte(line), 0o644); err != nil { //nolint:gosec // a marker file, not a secret
				g.logger.ErrorContext(ctx, "write stop file", "err", err)
			}
		}
	}
	if g.onTrip != "" {
		go g.runOnTrip(ctx, reason)
	}
}

// runOnTrip runs the operator's command once (evidence capture, a notification), with the reason in its env.
func (g *guard) runOnTrip(ctx context.Context, reason string) {
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, "/bin/sh", "-c", g.onTrip) //nolint:gosec // the operator's own --on_trip command
	cmd.Env = append(os.Environ(), "DRIVER_TRIP_REASON="+reason)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.logger.ErrorContext(ctx, "on_trip command failed", "err", err, "output", strings.TrimSpace(string(out)))
		return
	}
	g.logger.InfoContext(ctx, "on_trip command done")
}
