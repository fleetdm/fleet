// Command agentws-viz is a small development dashboard that visualizes the
// agent WebSocket connections held by a running Fleet server, live.
//
// It polls the server's /debug/agentws endpoint (which requires a global admin
// API token) and serves a self-refreshing dashboard page on localhost, showing
// each connection's host ID, remote address, uptime, last notification and
// counters, plus a running event log of connects, disconnects and
// notifications.
//
// Multi-instance deployments: /debug/agentws only reports the connections held
// by the instance that answers, and behind a load balancer each request may
// land on a different instance. The tool sends several concurrent requests per
// tick (see -instances) and buckets the responses by the instance_id each
// server reports, keeping the latest snapshot per instance; the dashboard
// shows one tab per instance. Instances that stop answering expire from the
// dashboard after a short while. Use -interval to poll less often on busy
// deployments.
//
// Usage:
//
//	go run ./tools/agentws-viz -server https://localhost:8080 -token $(fleetctl config get token) -insecure
//	go run ./tools/agentws-viz -server https://fleet.example.com -instances 6
//
// Then open http://localhost:3001.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// An instance that no request landed on for missedTicksStale ticks is shown as
// stale; after missedTicksDrop ticks it is dropped. Random load balancing can
// miss an instance for a few ticks; a real disappearance (deploy, scale-in)
// shows up after the drop threshold.
const (
	missedTicksStale = 3
	missedTicksDrop  = 15
)

type snapshotMeta struct {
	Enabled    bool   `json:"enabled"`
	InstanceID string `json:"instance_id"`
}

// instance is the latest snapshot seen from one Fleet server instance.
type instance struct {
	id     string
	body   []byte
	status int
	seenAt time.Time
}

type instanceInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Stale is set when the instance hasn't answered for a few ticks (it may
	// be gone; it expires after instanceTTL).
	Stale bool `json:"stale"`
}

// poller fans requests out through the load balancer and buckets responses
// by instance.
type poller struct {
	serverURL string
	token     string
	client    *http.Client
	fanout    int
	want      int
	interval  time.Duration

	mu        sync.Mutex
	instances map[string]*instance
	lastErr   string // last request error, cleared on any success
	disabled  bool   // server reports the transport disabled
	noID      bool   // server reports no instance_id (predates it)
}

func (p *poller) fetch(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.serverURL+"/debug/agentws", nil)
	if err != nil {
		p.setErr(err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	resp, err := p.client.Do(req)
	if err != nil {
		p.setErr(fmt.Errorf("fleet server unreachable: %w", err))
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.setErr(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		p.setErr(fmt.Errorf("fleet server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
		return
	}
	var meta snapshotMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		p.setErr(fmt.Errorf("decode snapshot: %w", err))
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastErr = ""
	p.disabled = !meta.Enabled
	if !meta.Enabled {
		return
	}
	id := meta.InstanceID
	if id == "" {
		// Server predates instance_id: everything lands in one bucket.
		p.noID = true
		id = "unknown"
	}
	inst, ok := p.instances[id]
	if !ok {
		inst = &instance{id: id}
		p.instances[id] = inst
		log.Printf("agentws-viz: discovered instance %s, %d/%d", short(id), len(p.instances), p.want)
	}
	inst.body, inst.status, inst.seenAt = body, resp.StatusCode, time.Now()
}

func (p *poller) setErr(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastErr = err.Error()
}

func (p *poller) run(ctx context.Context) {
	for ctx.Err() == nil {
		var wg sync.WaitGroup
		for range p.fanout {
			wg.Go(func() { p.fetch(ctx) })
		}
		wg.Wait()

		p.mu.Lock()
		for id, inst := range p.instances {
			if ttl := missedTicksDrop * p.interval; time.Since(inst.seenAt) > ttl {
				log.Printf("agentws-viz: instance %s not seen for %s, dropping", short(id), ttl)
				delete(p.instances, id)
			}
		}
		p.mu.Unlock()

		select {
		case <-ctx.Done():
		case <-time.After(p.interval):
		}
	}
}

func (p *poller) list() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]instanceInfo, 0, len(p.instances))
	for _, inst := range p.instances {
		out = append(out, instanceInfo{
			ID: inst.id, Label: short(inst.id),
			Stale: time.Since(inst.seenAt) > missedTicksStale*p.interval,
		})
	}
	// Stable tab order across refreshes.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	res := map[string]any{"want": p.want, "instances": out, "interval_ms": p.interval.Milliseconds()}
	if p.lastErr != "" {
		res["error"] = p.lastErr
	}
	if p.disabled {
		res["disabled"] = true
	}
	if p.noID {
		res["no_instance_id"] = true
	}
	return res
}

func (p *poller) snapshot(id string) ([]byte, int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id == "" && len(p.instances) == 1 {
		for _, inst := range p.instances {
			return inst.body, inst.status, true
		}
	}
	inst, ok := p.instances[id]
	if !ok {
		return nil, 0, false
	}
	return inst.body, inst.status, true
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func main() {
	var (
		serverURL = flag.String("server", "https://localhost:8080", "Fleet server URL")
		token     = flag.String("token", os.Getenv("FLEET_API_TOKEN"), "Fleet API token of a global admin (defaults to FLEET_API_TOKEN)")
		addr      = flag.String("addr", "127.0.0.1:3001", "address for the dashboard to listen on")
		insecure  = flag.Bool("insecure", false, "skip TLS certificate verification when talking to the Fleet server")
		instances = flag.Int("instances", 1, "number of Fleet server instances behind the load balancer; sizes the per-tick request fan-out")
		interval  = flag.Duration("interval", time.Second, "how often to poll the Fleet server (and refresh the dashboard)")
	)
	flag.Parse()

	if *token == "" {
		log.Fatal("missing Fleet API token: pass -token or set FLEET_API_TOKEN (get one with `fleetctl config get token` after `fleetctl login`)")
	}
	if *instances < 1 {
		log.Fatal("-instances must be at least 1")
	}
	if *interval < 100*time.Millisecond {
		log.Fatal("-interval must be at least 100ms")
	}
	*serverURL = strings.TrimRight(*serverURL, "/")

	client := fleethttp.NewClient(
		fleethttp.WithTimeout(10*time.Second),
		fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: *insecure, //nolint:gosec // local development tool, opt-in flag
		}),
	)

	// Oversample so every instance is very likely hit each tick even with
	// random/least-outstanding-requests balancing; misses are covered by the
	// TTL, and each instance is served on its own keep-alive connection.
	fanout := *instances
	if fanout > 1 {
		fanout *= 2
	}
	p := &poller{
		serverURL: *serverURL, token: *token, client: client,
		fanout: fanout, want: *instances, interval: *interval,
		instances: make(map[string]*instance),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))
	})
	mux.HandleFunc("/api/instances", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p.list())
	})
	// Serve the latest cached snapshot for an instance so the page needs no
	// token and no CORS/TLS handling.
	mux.HandleFunc("/api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		body, status, ok := p.snapshot(r.URL.Query().Get("instance"))
		if !ok {
			http.Error(w, "unknown instance", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})

	log.Printf("agentws-viz: watching %s/debug/agentws (%d instance(s), %d requests every %s)", *serverURL, *instances, fanout, *interval)
	log.Printf("agentws-viz: dashboard on http://%s", *addr)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err) //nolint:gocritic // exitAfterDefer: dev tool, nothing to clean up on exit
	}
}
