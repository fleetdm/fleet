package fleetmcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFleetClientLiveQueryWaits(t *testing.T) {
	// The variable configures the standalone binary (cmd/fleet-mcp reads it); an
	// embedded MCP shares the Fleet server's environment, where it means the
	// server's own period, so the package must only use what it's given.
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "999h")
	logger := slog.New(slog.DiscardHandler)

	fc, err := NewFleetClient(t.Context(), "https://fleet.example.com", "key", false, "", logger)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, fc.httpClient.Timeout)
	assert.Equal(t, defaultLiveQueryWait, fc.campaignWait)
	assert.Equal(t, defaultLiveQueryWait+liveQueryTimeoutMargin, fc.liveQueryClient.Timeout)
	assert.Same(t, fc.httpClient.Transport, fc.liveQueryClient.Transport, "live query client must share the default client's Transport")

	fc, err = NewFleetClient(t.Context(), "https://fleet.example.com", "key", false, "", logger,
		WithLiveQueryWaits(900*time.Second, 45*time.Second))
	require.NoError(t, err)
	assert.Equal(t, 900*time.Second, fc.campaignWait)
	assert.Equal(t, 45*time.Second+liveQueryTimeoutMargin, fc.liveQueryClient.Timeout)
}

func TestFleetClientUsesCallerToken(t *testing.T) {
	var gotAuth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 3})
	}))
	t.Cleanup(srv.Close)

	fc, err := NewFleetClient(t.Context(), srv.URL, "static-key", false, "", slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	_, err = fc.GetHostCount(t.Context())
	require.NoError(t, err)
	_, err = fc.GetHostCount(ContextWithFleetToken(t.Context(), "caller-token"))
	require.NoError(t, err)

	assert.Equal(t, []string{"Bearer static-key", "Bearer caller-token"}, gotAuth)
}

// The host name doesn't resolve, so both REST calls and the results websocket
// only work if they go through the custom dialer.
func TestFleetClientDialContextCoversRESTAndWebsocket(t *testing.T) {
	var mu sync.Mutex
	var restAuth, wsToken string
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/fleet/reports/run":
			mu.Lock()
			restAuth = r.Header.Get("Authorization")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"campaign": map[string]any{"id": 7}})
		case "/api/v1/fleet/results/websocket":
			conn, err := up.Upgrade(w, r, nil)
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()
			var auth struct {
				Data struct {
					Token string `json:"token"`
				} `json:"data"`
			}
			if !assert.NoError(t, conn.ReadJSON(&auth)) {
				return
			}
			mu.Lock()
			wsToken = auth.Data.Token
			mu.Unlock()
			var selectCampaign map[string]any
			if !assert.NoError(t, conn.ReadJSON(&selectCampaign)) {
				return
			}
			writeWSFrame(t, conn, "totals", map[string]any{"count": 1, "online": 1})
			writeWSFrame(t, conn, "result", map[string]any{
				"host": map[string]any{"id": 10, "hostname": "h10"},
				"rows": []map[string]string{{"answer": "42"}},
			})
		default:
			assert.Failf(t, "unexpected request", "%s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}
	fc, err := NewFleetClient(t.Context(), "http://fleet.internal", "", false, "", slog.New(slog.DiscardHandler),
		WithDialContext(dial), WithLiveQueryWaits(5*time.Second, 5*time.Second))
	require.NoError(t, err)

	ctx := ContextWithFleetToken(t.Context(), "caller-token")
	res, err := fc.runMultiHostCampaign(ctx, []uint{10}, "SELECT 1;", map[uint]Endpoint{10: {ID: 10, Name: "host-10"}})
	require.NoError(t, err)
	require.Len(t, res.Results, 1)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "Bearer caller-token", restAuth)
	assert.Equal(t, "caller-token", wsToken)
}

func listToolNames(t *testing.T, c *client.Client) []string {
	t.Helper()
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1"}
	initRes, err := c.Initialize(t.Context(), initReq)
	require.NoError(t, err)
	assert.NotContains(t, initRes.Instructions, "refresh_osquery_schema")

	tools, err := c.ListTools(t.Context(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		if tool.Name == "get_osquery_schema" {
			assert.NotContains(t, tool.Description, "refresh_osquery_schema")
		}
	}
	return names
}

func TestEmbeddedMCPServerHasNoSchemaRefresh(t *testing.T) {
	fc := newTestClient("http://127.0.0.1:1")
	c, err := client.NewInProcessClient(NewEmbeddedMCPServer(fc))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(t.Context()))

	names := listToolNames(t, c)
	assert.Contains(t, names, "run_live_query")
	assert.Contains(t, names, "get_osquery_schema")
	assert.NotContains(t, names, "refresh_osquery_schema")
	assert.Len(t, names, 19)
}

func TestEmbeddedHTTPHandlerCallsFleetAsCaller(t *testing.T) {
	var gotAuth string
	fleet := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 3})
	}))
	t.Cleanup(fleet.Close)

	fc, err := NewFleetClient(t.Context(), fleet.URL, "", false, "", slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	handler := NewEmbeddedHTTPHandler(NewEmbeddedMCPServer(fc), slog.New(slog.DiscardHandler))
	// Stands in for the Fleet server's wrapper, which authenticates the caller
	// and hands its token to the tools.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		handler.ServeHTTP(w, r.WithContext(ContextWithFleetToken(r.Context(), token)))
	}))
	t.Cleanup(srv.Close)

	c, err := client.NewStreamableHttpClient(srv.URL, transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer caller-token"}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(t.Context()))
	require.True(t, slices.Contains(listToolNames(t, c), "get_total_system_count"))

	req := mcp.CallToolRequest{}
	req.Params.Name = "get_total_system_count"
	res, err := c.CallTool(t.Context(), req)
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Equal(t, "Bearer caller-token", gotAuth)

	// Embedded mode has no standalone GET stream either.
	getReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	resp, err := srv.Client().Do(getReq)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// Fleet's router redirects unclean paths; the embedded client must not replay a
// call on whatever route the redirect names, while standalone keeps following
// redirects (e.g. http→https in front of Fleet).
func TestEmbeddedFleetClientDoesNotFollowRedirects(t *testing.T) {
	var redirectedTo bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/elsewhere" {
			redirectedTo = true
			_ = json.NewEncoder(w).Encode(map[string]int{"count": 1})
			return
		}
		http.Redirect(w, r, "/elsewhere", http.StatusMovedPermanently)
	}))
	t.Cleanup(srv.Close)
	logger := slog.New(slog.DiscardHandler)
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String())
	}

	embedded, err := NewFleetClient(t.Context(), "http://fleet.internal", "", false, "", logger, WithDialContext(dial))
	require.NoError(t, err)
	_, err = embedded.GetHostCount(t.Context())
	require.Error(t, err)
	assert.False(t, redirectedTo)

	standalone, err := NewFleetClient(t.Context(), srv.URL, "key", false, "", logger)
	require.NoError(t, err)
	_, err = standalone.GetHostCount(t.Context())
	require.NoError(t, err)
	assert.True(t, redirectedTo)
}
