package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// api is a small REST client with a context and a timeout on every call, and the status code in every result.
// (server/service.Client has neither; the driver uses it only for the live query websocket.)
type api struct {
	base   string
	token  string
	client *http.Client
}

func newAPI(base, token string, insecure bool, timeout time.Duration) *api {
	return &api{
		base:  strings.TrimRight(base, "/"),
		token: token,
		client: fleethttp.NewClient(
			fleethttp.WithTimeout(timeout),
			fleethttp.WithTLSClientConfig(&tls.Config{InsecureSkipVerify: insecure}), //nolint:gosec // opt-in flag for load-test instances with self-signed certs
		),
	}
}

// do calls Fleet as the driver's admin.
func (a *api) do(ctx context.Context, method, path string, query url.Values, body, out any) (int, error) {
	return a.doAs(ctx, a.token, method, path, query, body, out)
}

// doAs calls Fleet with another token, or none ("") for unauthenticated endpoints.
func (a *api) doAs(ctx context.Context, token, method, path string, query url.Values, body, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("%s %s: encode body: %w", method, path, err)
		}
		rd = bytes.NewReader(b)
	}
	u := a.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd) //nolint:gosec // the operator chooses the Fleet URL
	if err != nil {
		return 0, fmt.Errorf("%s %s: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.client.Do(req) //nolint:gosec // the operator chooses the Fleet URL
	if err != nil {
		return 0, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return resp.StatusCode, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("%s %s: decode: %w", method, path, err)
		}
	}
	return resp.StatusCode, nil
}

type hostInfo struct {
	ID       uint   `json:"id"`
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	FleetID  *uint  `json:"fleet_id"`
	Status   string `json:"status"`
}

// hostCache keeps the instance's host list. Listing 10k hosts takes ~20 pages and up to a minute on a loaded
// instance, so it's filled before the workload starts and refreshed in the background, never inside an action. A
// failed refresh keeps the old list.
type hostCache struct {
	api    *api
	every  time.Duration
	logger *slog.Logger

	mu    sync.RWMutex
	hosts []hostInfo
}

// start fills the list and keeps refreshing it until ctx ends.
func (c *hostCache) start(ctx context.Context) error {
	if err := c.refresh(ctx); err != nil {
		return fmt.Errorf("host list: %w", err)
	}
	go func() {
		t := time.NewTicker(c.every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := c.refresh(ctx); err != nil && ctx.Err() == nil {
					c.logger.WarnContext(ctx, "host list refresh failed, keeping the old list", "err", err)
				}
			}
		}
	}()
	return nil
}

func (c *hostCache) refresh(ctx context.Context) error {
	start := time.Now()
	var all []hostInfo
	for page := 0; ; page++ {
		var resp struct {
			Hosts []hostInfo `json:"hosts"`
		}
		q := url.Values{
			"page": {strconv.Itoa(page)}, "per_page": {"500"}, "order_key": {"id"},
			"disable_issues": {"true"},
		}
		if _, err := c.api.do(ctx, http.MethodGet, "/api/latest/fleet/hosts", q, nil, &resp); err != nil {
			return err
		}
		all = append(all, resp.Hosts...)
		if len(resp.Hosts) < 500 {
			break
		}
	}
	c.mu.Lock()
	c.hosts = all
	c.mu.Unlock()
	c.logger.InfoContext(ctx, "host list refreshed", "hosts", len(all), "took", time.Since(start).Round(time.Second))
	return nil
}

// moved records a transfer until the next refresh.
func (c *hostCache) moved(ids []uint, fleetID uint) {
	set := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	hosts := slices.Clone(c.hosts) // pick may still hold the old slice
	for i := range hosts {
		if _, ok := set[hosts[i].ID]; ok {
			hosts[i].FleetID = new(fleetID)
		}
	}
	c.hosts = hosts
}

// pick returns up to n distinct random hosts, only online ones when online is set.
func (c *hostCache) pick(n int, online bool) []hostInfo {
	c.mu.RLock()
	all := c.hosts
	c.mu.RUnlock()
	pool := all
	if online {
		pool = make([]hostInfo, 0, len(all))
		for _, h := range all {
			if h.Status == "online" {
				pool = append(pool, h)
			}
		}
	}
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]hostInfo, 0, n)
	for _, i := range randPerm(len(pool))[:n] {
		out = append(out, pool[i])
	}
	return out
}
