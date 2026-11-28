package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// reading is a metric's value over one window. none means the window had no data (no requests, no datapoints yet),
// which is neither ok nor a breach.
type reading struct {
	value  float64
	none   bool
	detail map[string]any
}

// source reads one metric. lag is how far behind now its data settles (CloudWatch publishes a minute or more
// late), so windows end at now minus lag.
type source interface {
	read(ctx context.Context, from, to time.Time) (reading, error)
	lag() time.Duration
}

var errSourceUnavailable = errors.New("not available")

// metricSources builds the source for a metric name. The names follow FORMAT.md's shared metric names where one
// exists.
type metricSources struct {
	guard *guard
	api   *api
	aws   *awsSources // nil without -lt
}

func metricNames() []string {
	return []string{
		"healthz.p99", "hosts.online_drop",
		"fleet.memory.max", "fleet.cpu.max", "fleet.tasks_missing", "fleet.task_stops", "alb.5xx_pct",
		"writer.cpu", "reader.cpu",
		"host.distributed_write.p99", "osquery.config.p99", "orbit.config.p99", "checkin.errors",
	}
}

// logURIs are the requests whose p99 the log-based metrics read.
var logURIs = map[string]string{
	"host.distributed_write.p99": "/api/osquery/distributed/write",
	"osquery.config.p99":         "/api/osquery/config",
	"orbit.config.p99":           "/api/fleet/orbit/config",
}

func (ms *metricSources) source(name string) (source, unit, error) {
	switch name {
	case "healthz.p99":
		return &healthzSource{g: ms.guard}, unitSeconds, nil
	case "hosts.online_drop":
		return &onlineDropSource{api: ms.api}, unitPercent, nil
	}
	if !slices.Contains(metricNames(), name) {
		return nil, 0, fmt.Errorf("unknown metric %q (known: %s)", name, strings.Join(metricNames(), ", "))
	}
	if ms.aws == nil {
		return nil, 0, fmt.Errorf("%w without -lt (it reads AWS)", errSourceUnavailable)
	}
	return ms.aws.source(name)
}

// healthzSource is the p99 of the guard's own /healthz readings (one a minute); a failed reading counts as the
// reading's timeout.
type healthzSource struct{ g *guard }

func (s *healthzSource) lag() time.Duration { return 0 }

func (s *healthzSource) read(_ context.Context, from, to time.Time) (reading, error) {
	var took []float64
	failed := 0
	for _, h := range s.g.healthzReadings(from, to) {
		took = append(took, h.took.Seconds())
		if !h.ok {
			failed++
		}
	}
	if len(took) == 0 {
		return reading{none: true}, nil
	}
	slices.Sort(took)
	return reading{value: quantile(took, 0.99), detail: map[string]any{"readings": len(took), "failed": failed}}, nil
}

// quantile is the nearest-rank q-quantile of sorted values.
func quantile(sorted []float64, q float64) float64 {
	i := int(q*float64(len(sorted))+0.5) - 1
	return sorted[max(0, min(i, len(sorted)-1))]
}

// onlineDropSource is how far the online host count is below its highest reading in the window, in percent: hosts
// dropping off is the first sign of an agent-facing outage that healthz doesn't show.
type onlineDropSource struct {
	api *api

	mu   sync.Mutex
	seen []countAt
}

type countAt struct {
	at time.Time
	n  int
}

func (s *onlineDropSource) lag() time.Duration { return 0 }

func (s *onlineDropSource) read(ctx context.Context, from, to time.Time) (reading, error) {
	var resp struct {
		OnlineCount int `json:"online_count"`
	}
	if _, err := s.api.do(ctx, http.MethodGet, "/api/latest/fleet/host_summary", nil, nil, &resp); err != nil {
		return reading{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, countAt{at: to, n: resp.OnlineCount})
	s.seen = slices.DeleteFunc(s.seen, func(c countAt) bool { return c.at.Before(from) })
	base := 0
	for _, c := range s.seen {
		base = max(base, c.n)
	}
	if base == 0 {
		return reading{none: true}, nil
	}
	drop := 100 * float64(base-resp.OnlineCount) / float64(base)
	return reading{value: drop, detail: map[string]any{"online": resp.OnlineCount, "baseline": base}}, nil
}
