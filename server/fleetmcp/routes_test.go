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
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const liveQuerySQL = "SELECT version FROM os_version;"

// Drives every argument mode of every embedded tool against a fake Fleet, so a
// tool's ToolRoutes entry is exactly the routes it can call: a missing route
// would let a restricted user reach a tool that fails partway, an extra one
// would hide a tool the user can use.
func TestToolRoutesMatchTools(t *testing.T) {
	fake, fc := newFakeFleet(t)
	c := newInProcessMCPClient(t, NewEmbeddedMCPServer(fc))
	tools := listToolNames(t, c)

	cve := "CVE-2024-0001"
	modes := map[string][]map[string]any{
		"get_aggregate_platforms": {nil},
		"get_endpoints":           {nil, {"fleet": "Workstations", "label": "Engineering"}},
		"get_fleets":              {nil},
		// "alpha" matches one hostname; no hostname matches a UUID, so that
		// falls back to the identifier route.
		"get_host":                 {{"host_id": "42"}, {"identifier": "alpha"}, {"identifier": "uuid-42"}},
		"get_host_policies":        {{"host_id": "42"}, {"identifier": "uuid-42"}},
		"get_host_users":           {{"host_id": "42"}, {"host_identifier": "uuid-42"}},
		"get_labels":               {nil},
		"get_osquery_schema":       {{"tables": "os_version"}},
		"get_policies":             {nil},
		"get_policy_compliance":    {{"policy_id": "3"}, {"policy_id": "3", "fleet": "Workstations"}},
		"get_policy_hosts":         {{"policy_id": "3"}, {"policy_id": "3", "response": "failing", "fleet": "Workstations", "label": "Engineering"}},
		"get_queries":              {nil},
		"get_software":             {{"fleet": "Workstations"}, {"host_id": "42"}, {"host_identifier": "uuid-42"}},
		"get_total_system_count":   {nil},
		"get_vetted_queries":       {nil},
		"get_vulnerability_hosts":  {{"cve_id": cve, "fleet": "Workstations"}},
		"get_vulnerability_impact": {{"cve_id": cve}},
		"prepare_live_query": {
			{"host_ids": "42"}, {"hostnames": "uuid-42"}, {"fleet": "Workstations", "label": "Engineering"}, {"cve_id": cve},
		},
		// One target runs on the host's REST endpoint, more start a campaign.
		"run_live_query": {
			{"sql": liveQuerySQL, "host_ids": "42"},
			{"sql": liveQuerySQL, "hostnames": "uuid-42"},
			{"sql": liveQuerySQL, "fleet": "Workstations", "label": "Engineering"},
			{"sql": liveQuerySQL, "cve_id": cve},
		},
	}
	require.ElementsMatch(t, tools, slices.Collect(maps.Keys(modes)), "every embedded tool must be driven")
	require.ElementsMatch(t, tools, slices.Collect(maps.Keys(ToolRoutes)), "every embedded tool needs a ToolRoutes entry")

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			for _, args := range modes[tool] {
				res := callTool(t, t.Context(), c, tool, args)
				assert.False(t, res.IsError, "%v: %v", args, res.Content)
			}
			hit, unmatched := fake.take()
			assert.Empty(t, unmatched, "requests to routes no tool declares")
			assert.ElementsMatch(t, ToolRoutes[tool], slices.Collect(maps.Keys(hit)))
		})
	}
}

func TestEmbeddedToolsFollowRouteFilter(t *testing.T) {
	fake, fc := newFakeFleet(t)
	s := NewEmbeddedMCPServer(fc)
	s.AddTool(mcp.NewTool("undeclared"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ran"), nil
	})
	c := newInProcessMCPClient(t, s)
	assert.Len(t, listToolNames(t, c), 20, "without a filter every tool is listed")

	granted := []Route{{"GET", "/api/v1/fleet/fleets"}, {"GET", "/api/v1/fleet/hosts/count"}}
	ctx := ContextWithRouteFilter(t.Context(), func(r Route) bool { return slices.Contains(granted, r) })

	list, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	var names []string
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	assert.ElementsMatch(t, []string{"get_fleets", "get_osquery_schema", "get_total_system_count", "get_vetted_queries"}, names)

	res := callTool(t, ctx, c, "get_endpoints", nil)
	require.True(t, res.IsError)
	text := res.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "GET /api/v1/fleet/hosts, GET /api/v1/fleet/hosts/:id, GET /api/v1/fleet/labels, GET /api/v1/fleet/labels/:id/hosts.")
	assert.NotContains(t, text, "/hosts/count")
	res = callTool(t, ctx, c, "undeclared", nil)
	require.True(t, res.IsError, "a tool missing from ToolRoutes must not run for a restricted user")
	hit, unmatched := fake.take()
	assert.Empty(t, hit, "a hidden tool must not reach Fleet")
	assert.Empty(t, unmatched)

	res = callTool(t, ctx, c, "get_total_system_count", nil)
	assert.False(t, res.IsError, res.Content)
}

// mcp-go runs a task-augmented tools/call without the tool handler middlewares,
// so a task tool would skip the route filter's call guard.
func TestEmbeddedToolsAreNotTasks(t *testing.T) {
	_, fc := newFakeFleet(t)
	c := newInProcessMCPClient(t, NewEmbeddedMCPServer(fc))
	require.Len(t, listToolNames(t, c), len(ToolRoutes))
	res, err := c.ListTools(t.Context(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	for _, tool := range res.Tools {
		assert.True(t, tool.Execution == nil || tool.Execution.TaskSupport == mcp.TaskSupportForbidden, tool.Name)
	}
}

func newFakeFleet(t *testing.T) (*routeRecorder, *FleetClient) {
	fake := &routeRecorder{t: t, hit: map[Route]bool{}}
	for _, routes := range ToolRoutes {
		for _, r := range routes {
			if !slices.Contains(fake.routes, r) {
				fake.routes = append(fake.routes, r)
			}
		}
	}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	fc, err := NewFleetClient(t.Context(), "http://fleet.internal", "", false, "", slog.New(slog.DiscardHandler),
		WithDialContext(dial), WithLiveQueryWaits(5*time.Second, 5*time.Second))
	require.NoError(t, err)
	return fake, fc
}

func newInProcessMCPClient(t *testing.T, s *server.MCPServer) *client.Client {
	c, err := client.NewInProcessClient(s)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(t.Context()))
	return c
}

func callTool(t *testing.T, ctx context.Context, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(ctx, req)
	require.NoError(t, err, name)
	require.NotEmpty(t, res.Content, name)
	return res
}

// routeRecorder serves the routes in ToolRoutes and records which ones the
// tools reach.
type routeRecorder struct {
	t         *testing.T
	routes    []Route
	mu        sync.Mutex
	hit       map[Route]bool
	unmatched []string
}

func (f *routeRecorder) take() (map[Route]bool, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	hit, unmatched := f.hit, f.unmatched
	f.hit, f.unmatched = map[Route]bool{}, nil
	return hit, unmatched
}

func (f *routeRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The results websocket authenticates its own messages rather than going
	// through the endpoint allowlist, so it isn't a tool route.
	if r.URL.Path == "/api/v1/fleet/results/websocket" {
		f.serveCampaignResults(w, r)
		return
	}
	route, params, ok := f.match(r.Method, r.URL.EscapedPath())
	f.mu.Lock()
	if ok {
		f.hit[route] = true
	} else {
		f.unmatched = append(f.unmatched, r.Method+" "+r.URL.Path)
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

// match resolves a request the way Fleet's router would, with a literal
// segment winning over a parameter (hosts/count over hosts/:id).
func (f *routeRecorder) match(method, path string) (Route, map[string]string, bool) {
	segs := strings.Split(path, "/")
	var best Route
	var bestParams map[string]string
	bestLiterals := -1
routes:
	for _, r := range f.routes {
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
		require.Error(t, err, bad)
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
