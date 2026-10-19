package fleetmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Drives every embedded tool against a Fleet that serves only EmbeddedRoutes,
// so a tool can't call a route the list doesn't grant and the list can't grant
// a route no tool calls.
func TestEmbeddedRoutesMatchTools(t *testing.T) {
	fake := &routeRecorder{t: t, hit: map[Route]bool{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	fc, err := NewFleetClient(t.Context(), "http://fleet.internal", "", false, "", slog.New(slog.DiscardHandler),
		WithDialContext(dial), WithLiveQueryWaits(5*time.Second, 5*time.Second))
	require.NoError(t, err)

	c, err := client.NewInProcessClient(NewEmbeddedMCPServer(fc))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(t.Context()))
	tools := listToolNames(t, c)

	called := map[string]bool{}
	for _, call := range []struct {
		tool string
		args map[string]any
	}{
		{"get_endpoints", nil},
		{"get_endpoints", map[string]any{"label": "Engineering"}},
		{"get_host", map[string]any{"host_id": "42"}},
		// No hostname matches a UUID, so this falls back to the identifier route.
		{"get_host", map[string]any{"identifier": "uuid-42"}},
		{"get_host_policies", map[string]any{"identifier": "uuid-42"}},
		{"get_host_users", map[string]any{"host_id": "42"}},
		{"get_total_system_count", nil},
		{"get_aggregate_platforms", nil},
		{"get_fleets", nil},
		{"get_labels", nil},
		{"get_queries", nil},
		{"get_vetted_queries", nil},
		{"get_osquery_schema", map[string]any{"tables": "os_version"}},
		{"prepare_live_query", map[string]any{"host_ids": "42"}},
		{"run_live_query", map[string]any{"sql": "SELECT version FROM os_version;", "host_ids": "42"}},
		{"run_live_query", map[string]any{"sql": "SELECT version FROM os_version;", "host_ids": "42,43"}},
		{"get_policies", nil},
		{"get_policy_compliance", map[string]any{"policy_id": "3"}},
		{"get_policy_compliance", map[string]any{"policy_id": "3", "fleet": "Workstations"}},
		{"get_policy_hosts", map[string]any{"policy_id": "3", "response": "failing"}},
		{"get_vulnerability_impact", map[string]any{"cve_id": "CVE-2024-0001"}},
		{"get_vulnerability_hosts", map[string]any{"cve_id": "CVE-2024-0001"}},
		{"get_software", nil},
		{"get_software", map[string]any{"host_id": "42"}},
	} {
		req := mcp.CallToolRequest{}
		req.Params.Name = call.tool
		req.Params.Arguments = call.args
		res, err := c.CallTool(t.Context(), req)
		require.NoError(t, err, call.tool)
		assert.False(t, res.IsError, "%s %v: %v", call.tool, call.args, res.Content)
		called[call.tool] = true
	}
	assert.ElementsMatch(t, tools, slices.Collect(maps.Keys(called)), "every embedded tool must be driven")

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Empty(t, fake.unlisted, "requests to routes missing from EmbeddedRoutes")
	var unused []Route
	for _, r := range EmbeddedRoutes {
		if !fake.hit[r] {
			unused = append(unused, r)
		}
	}
	assert.Empty(t, unused, "EmbeddedRoutes no tool calls")
}

type routeRecorder struct {
	t        *testing.T
	mu       sync.Mutex
	hit      map[Route]bool
	unlisted []string
}

func (f *routeRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The results websocket authenticates its own messages, so the route check
	// the list feeds doesn't apply to it.
	if r.URL.Path == "/api/v1/fleet/results/websocket" {
		f.serveCampaignResults(w, r)
		return
	}
	route, params, ok := matchEmbeddedRoute(r.Method, r.URL.EscapedPath())
	f.mu.Lock()
	if ok {
		f.hit[route] = true
	} else {
		f.unlisted = append(f.unlisted, r.Method+" "+r.URL.Path)
	}
	f.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	body, found := fakeFleetResponse(route, params, r.URL.Query())
	if !found {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (f *routeRecorder) serveCampaignResults(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if !assert.NoError(f.t, err) {
		return
	}
	defer conn.Close()
	for range 2 { // auth, select_campaign
		var msg map[string]any
		if !assert.NoError(f.t, conn.ReadJSON(&msg)) {
			return
		}
	}
	writeWSFrame(f.t, conn, "status", map[string]any{"status": "finished"})
}

// matchEmbeddedRoute resolves a request the way Fleet's router would, with a
// literal segment winning over a parameter (hosts/count over hosts/:id).
func matchEmbeddedRoute(method, path string) (Route, map[string]string, bool) {
	segs := strings.Split(path, "/")
	var best Route
	var bestParams map[string]string
	bestLiterals := -1
routes:
	for _, r := range EmbeddedRoutes {
		tpl := strings.Split(r.Path, "/")
		if r.Method != method || len(tpl) != len(segs) {
			continue
		}
		params := map[string]string{}
		literals := 0
		for i, s := range tpl {
			switch {
			case s == segs[i]:
				literals++
			case strings.HasPrefix(s, ":") && segs[i] != "":
				params[s[1:]] = segs[i]
			default:
				continue routes
			}
		}
		if literals > bestLiterals {
			best, bestParams, bestLiterals = r, params, literals
		}
	}
	return best, bestParams, bestLiterals >= 0
}

var fakeHosts = []map[string]any{
	{"id": 42, "uuid": "uuid-42", "hostname": "alpha.local", "platform": "darwin", "status": "online", "team_id": 1, "team_name": "Workstations"},
	{"id": 43, "uuid": "uuid-43", "hostname": "beta.local", "platform": "darwin", "status": "online", "team_id": 1, "team_name": "Workstations"},
}

// fakeFleetResponse returns the minimal body each route needs for the tools to
// carry on to their next call.
func fakeFleetResponse(route Route, params map[string]string, query url.Values) (any, bool) {
	policy := map[string]any{"id": 3, "name": "Disk encryption", "passing_host_count": 1, "failing_host_count": 1}
	hostDetail := func(match func(h map[string]any) bool) (any, bool) {
		for _, h := range fakeHosts {
			if match(h) {
				d := maps.Clone(h)
				d["labels"] = []any{map[string]any{"id": 7, "name": "Engineering"}}
				d["users"] = []any{map[string]any{"uid": 501, "username": "alice"}}
				d["policies"] = []any{map[string]any{"id": 3, "name": "Disk encryption", "response": "fail"}}
				return map[string]any{"host": d}, true
			}
		}
		return nil, false
	}

	switch route.Path {
	case "/api/v1/fleet/fleets":
		return map[string]any{"teams": []any{map[string]any{"id": 1, "name": "Workstations"}}}, true
	case "/api/v1/fleet/fleets/:id/policies", "/api/v1/fleet/global/policies":
		return map[string]any{"policies": []any{policy}}, true
	case "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id", "/api/v1/fleet/global/policies/:id":
		return map[string]any{"policy": policy}, true
	case "/api/v1/fleet/host_summary":
		return map[string]any{"totals_hosts_count": 2, "platforms": []any{map[string]any{"platform": "darwin", "hosts_count": 2}}}, true
	case "/api/v1/fleet/hosts", "/api/v1/fleet/labels/:id/hosts":
		// Like Fleet's search, the query doesn't match UUIDs.
		hosts := []any{}
		for _, h := range fakeHosts {
			if strings.Contains(h["hostname"].(string), query.Get("query")) {
				hosts = append(hosts, h)
			}
		}
		return map[string]any{"hosts": hosts}, true
	case "/api/v1/fleet/hosts/count":
		return map[string]any{"count": len(fakeHosts)}, true
	case "/api/v1/fleet/hosts/:id":
		return hostDetail(func(h map[string]any) bool { return params["id"] == fmt.Sprint(h["id"]) })
	case "/api/v1/fleet/hosts/identifier/:identifier":
		return hostDetail(func(h map[string]any) bool {
			return params["identifier"] == h["uuid"] || params["identifier"] == h["hostname"]
		})
	case "/api/v1/fleet/hosts/:id/software":
		return map[string]any{"software": []any{map[string]any{"id": 9, "name": "Chrome", "source": "apps"}}}, true
	case "/api/v1/fleet/labels":
		return map[string]any{"labels": []any{map[string]any{"id": 7, "name": "Engineering"}}}, true
	case "/api/v1/fleet/reports":
		return map[string]any{"queries": []any{map[string]any{"id": 5, "name": "Uptime", "query": "SELECT * FROM uptime;"}}}, true
	case "/api/v1/fleet/software/titles":
		return map[string]any{"software_titles": []any{map[string]any{"id": 9, "name": "Chrome", "source": "apps"}}}, true
	case "/api/v1/fleet/software/titles/:id":
		return map[string]any{"software_title": map[string]any{"id": 9, "versions": []any{map[string]any{"id": 11}}}}, true
	case "/api/v1/fleet/hosts/:id/query":
		return map[string]any{"status": "online", "rows": []any{map[string]any{"version": "15.0"}}}, true
	case "/api/v1/fleet/reports/run":
		return map[string]any{"campaign": map[string]any{"id": 77}}, true
	}
	return nil, false
}

func TestPathSegment(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../123/recovery_lock_password", "a/b", `a\b`, "123/encryption_key"} {
		_, err := pathSegment("host identifier", bad)
		assert.Error(t, err, bad)
	}
	for in, want := range map[string]string{
		"host-1.local":                         "host-1.local",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"..foo":                                "..foo",
		"alice's mac":                          "alice%27s%20mac",
		"host?x=1#y":                           "host%3Fx=1%23y",
	} {
		got, err := pathSegment("host identifier", in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}
}

func TestRouteForLogRedactsIdentifiers(t *testing.T) {
	assert.Equal(t, "/api/v1/fleet/hosts/identifier/:identifier", routeForLog("/api/v1/fleet/hosts/identifier/alice-macbook?populate_policies=true"))
	assert.Equal(t, "/api/v1/fleet/hosts", routeForLog("/api/v1/fleet/hosts?query=alice@example.com"))
}
