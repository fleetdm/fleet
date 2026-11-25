package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// guard decides whether the workload runs. The stop marker pauses it (whoever wrote it: this driver, a watcher, a
// person), and the driver's own wires write it on a trip. Recovery is by hand: the workload resumes when someone
// removes the marker. Only Fleet's own symptoms trip; database load alone doesn't (it's watched elsewhere and at
// this population it's high through every vulnerabilities run).
type guard struct {
	stop      stopMarker // nil: no marker, trips pause until the driver restarts
	onTrip    string     // shell command run once per trip
	notifyURL string     // webhook (Slack incoming-webhook shape) told about each trip
	api       *api
	events    *eventLog
	logger    *slog.Logger

	mu        sync.Mutex
	tripped   string
	isPaused  bool
	markerErr bool   // the last marker read failed (logged once)
	unmarked  bool   // a trip couldn't write the marker, so only a restart resumes
	recent    []bool // the last outcomes, true = broken on the server's side
	hzFails   int
	hz        []healthzReading // the last two hours, for the healthz.p99 check
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

// paused reports whether actions should be skipped now, and why. It writes pause and resume events on changes. A
// marker that can't be read keeps the last state.
func (g *guard) paused(ctx context.Context) (bool, string) {
	var line string
	var exists, readOK bool
	if g.stop != nil {
		var err error
		line, exists, err = g.stop.get(ctx)
		readOK = err == nil
		g.mu.Lock()
		if err != nil && !g.markerErr {
			g.logger.WarnContext(ctx, "can't read the stop marker, keeping the last state", "err", err)
		}
		g.markerErr = err != nil
		g.mu.Unlock()
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	reason := ""
	switch {
	case g.stop != nil && !readOK && g.isPaused:
		return true, "stop marker unreadable, still paused"
	case exists:
		reason = "stop file: " + line
	case g.stop != nil && readOK && !g.unmarked:
		g.tripped = "" // the marker was removed: recovered by hand
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
	where := "(none: restart the driver to resume)"
	if g.stop != nil {
		where = g.stop.String()
		line := fmt.Sprintf("%s driver: %s", time.Now().UTC().Format("2006-01-02T15:04Z"), reason)
		if err := g.stop.create(ctx, line); err != nil {
			g.logger.ErrorContext(ctx, "write the stop marker; the workload stays paused until the driver restarts", "err", err)
			g.mu.Lock()
			g.unmarked = true
			g.mu.Unlock()
		}
	}
	g.logger.ErrorContext(ctx, "trip: workload paused until the stop marker is removed", "reason", reason, "stop_marker", where)
	if g.onTrip != "" {
		go g.runOnTrip(ctx, reason)
	}
	if g.notifyURL != "" {
		go g.notify(ctx, reason, where)
	}
}

// notify posts the trip to a webhook that takes {"text": ...} (a Slack incoming webhook).
func (g *guard) notify(ctx context.Context, reason, where string) {
	nctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	text := fmt.Sprintf("Load-test activity driver tripped on %s: %s. The workload is paused until %s is removed.",
		g.api.base, reason, where)
	req, err := http.NewRequestWithContext(nctx, http.MethodPost, g.notifyURL, bytes.NewReader(mustJSON(map[string]string{"text": text})))
	if err != nil {
		g.logger.ErrorContext(ctx, "trip notification", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second)).Do(req) //nolint:gosec // the operator's own webhook
	if err != nil {
		g.logger.ErrorContext(ctx, "trip notification", "err", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		g.logger.ErrorContext(ctx, "trip notification", "status", resp.StatusCode)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
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
