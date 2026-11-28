package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/service"
)

// liveQuery runs an ad-hoc live query the way the UI and fleetctl do (a campaign plus the results websocket), on a
// few online hosts or on all hosts, and waits until every online target answered or the timeout.
type liveQuery struct {
	all      bool
	minHosts int
	maxHosts int
	timeout  time.Duration
	queries  []string
}

// Everyday questions an admin asks a few hosts. osquery-perf answers any query, so the SQL shapes the request, not
// the hosts' work.
var defaultLiveQueries = []string{
	"SELECT * FROM os_version;",
	"SELECT * FROM system_info;",
	"SELECT name, version FROM programs;",
	"SELECT name, bundle_short_version FROM apps WHERE name LIKE '%Chrome%';",
	"SELECT * FROM uptime;",
	"SELECT username, uid FROM users WHERE uid >= 500;",
	"SELECT name, path, pid FROM processes WHERE name LIKE '%agent%';",
	"SELECT * FROM disk_encryption;",
}

func newLiveQuery(p *params) (verb, error) {
	v := &liveQuery{queries: defaultLiveQueries}
	target, err := p.str("target", "hosts")
	if err != nil {
		return nil, err
	}
	switch target {
	case "all":
		v.all = true
	case "hosts":
		if v.minHosts, v.maxHosts, err = p.intRange("hosts", 1, 50); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("with.target %q: hosts or all", target)
	}
	if v.timeout, err = p.duration("timeout", 60*time.Second); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *liveQuery) setup(context.Context, *env) error { return nil }

func (v *liveQuery) run(ctx context.Context, e *env) result {
	var labels, idents []string
	detail := map[string]any{}
	if v.all {
		labels = []string{"All Hosts"}
		detail["target"] = "all"
	} else {
		hs := e.hosts.pick(v.minHosts+randIntN(v.maxHosts-v.minHosts+1), true)
		if len(hs) == 0 {
			return result{skip: "no online hosts"}
		}
		for _, h := range hs {
			idents = append(idents, h.UUID)
		}
		detail["target"] = len(idents)
	}

	c, err := service.NewClient(e.url, e.insecure, "", "")
	if err != nil {
		return result{err: err, detail: detail}
	}
	c.SetToken(e.api.token)
	qctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()
	start := time.Now()
	h, err := c.LiveQueryWithContext(qctx, v.queries[randIntN(len(v.queries))], nil, labels, idents)
	if err != nil {
		return result{err: err, detail: detail}
	}
	defer drainLiveQuery(cancel, h)

	var rows, hostErrors int
	var lastErr error
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case res := <-h.Results():
			if _, ok := detail["first_result_ms"]; !ok {
				detail["first_result_ms"] = time.Since(start).Milliseconds()
			}
			rows += len(res.Rows)
			if res.Error != nil {
				hostErrors++
			}
		case err := <-h.Errors():
			lastErr = err
		case <-tick.C:
			st, tt := h.Status(), h.Totals()
			if st == nil || tt == nil {
				continue
			}
			detail["online"], detail["responded"] = tt.Online, st.ActualResults
			if st.ActualResults >= tt.Online {
				detail["rows"], detail["host_errors"] = rows, hostErrors
				return result{status: 200, detail: detail}
			}
		case <-qctx.Done():
			detail["rows"], detail["host_errors"] = rows, hostErrors
			if ctx.Err() != nil { // the driver is stopping
				return result{err: ctx.Err(), detail: detail}
			}
			responded, _ := detail["responded"].(uint)
			if responded == 0 {
				if lastErr == nil {
					lastErr = errors.New("no online target answered")
				}
				return result{err: fmt.Errorf("live query: %w within %s", lastErr, v.timeout), detail: detail}
			}
			// Some online targets never answered: the action worked, its outcome is late.
			return result{status: 200, slow: true, detail: detail}
		}
	}
}

// drainLiveQuery closes the campaign and reads until the websocket goroutine exits; it blocks on unbuffered sends,
// so it would leak (with its connection) if nobody read.
func drainLiveQuery(cancel context.CancelFunc, h *service.LiveQueryResultsHandler) {
	cancel()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-h.Done():
			return
		case <-h.Results():
		case <-h.Errors():
		case <-timeout:
			return
		}
	}
}
