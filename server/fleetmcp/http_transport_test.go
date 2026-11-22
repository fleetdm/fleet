package fleetmcp

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMCPAuthToken = "test-mcp-auth-token-0123456789abcdef" //nolint:gosec // dummy token for tests

const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`

func newTestHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("FLEET_MCP_SCHEMA_REFRESH_DISABLE", "1")
	// The tools these tests call are served from the in-memory schema, so the
	// Fleet client is never dialed.
	logger := slog.New(slog.DiscardHandler)
	fleetClient, err := NewFleetClient(t.Context(), "http://127.0.0.1:1", "unused", false, "", logger)
	require.NoError(t, err)
	srv := httptest.NewServer(NewHTTPHandler(SetupMCPServer(t.Context(), fleetClient), testMCPAuthToken, logger))
	t.Cleanup(srv.Close)
	return srv
}

func exerciseMCPClient(t *testing.T, c *client.Client) {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, c.Start(ctx))

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "fleet-mcp-test", Version: "1"}
	initRes, err := c.Initialize(ctx, initReq)
	require.NoError(t, err)
	assert.Equal(t, "fleet-mcp", initRes.ServerInfo.Name)

	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	assert.Contains(t, names, "get_osquery_schema")
	assert.Contains(t, names, "run_live_query")

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = "get_osquery_schema"
	callReq.Params.Arguments = map[string]any{"tables": "os_version"}
	callRes, err := c.CallTool(ctx, callReq)
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	require.NotEmpty(t, callRes.Content)
	text, ok := mcp.AsTextContent(callRes.Content[0])
	require.True(t, ok)
	assert.Contains(t, text.Text, `"os_version"`)
}

func TestHTTPHandler_StreamableHTTP(t *testing.T) {
	srv := newTestHTTPServer(t)
	c, err := client.NewStreamableHttpClient(srv.URL+"/mcp",
		transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + testMCPAuthToken}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	exerciseMCPClient(t, c)
}

func TestHTTPHandler_SSE(t *testing.T) {
	srv := newTestHTTPServer(t)
	c, err := client.NewSSEMCPClient(srv.URL+"/sse",
		client.WithHeaders(map[string]string{"Authorization": "Bearer " + testMCPAuthToken}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	exerciseMCPClient(t, c)
}

func TestHTTPHandler_StreamableHTTPIsStateless(t *testing.T) {
	srv := newTestHTTPServer(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/mcp", strings.NewReader(initializeBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testMCPAuthToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get(server.HeaderKeySessionID))

	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/mcp", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testMCPAuthToken)
	req.Header.Set("Accept", "text/event-stream")
	resp, err = srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestHTTPHandler_RequiresBearerToken(t *testing.T) {
	srv := newTestHTTPServer(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/mcp"},
		{http.MethodGet, "/mcp"},
		{http.MethodDelete, "/mcp"},
		{http.MethodGet, "/sse"},
		{http.MethodPost, "/message"},
	}
	for _, route := range routes {
		for _, authz := range []string{"", "Bearer wrong"} {
			t.Run(route.method+route.path+"/"+authz, func(t *testing.T) {
				req, err := http.NewRequestWithContext(t.Context(), route.method, srv.URL+route.path, strings.NewReader(initializeBody))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				if authz != "" {
					req.Header.Set("Authorization", authz)
				}
				resp, err := srv.Client().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})
		}
	}
}

func TestHTTPHandler_HealthzAndUnknownPathsSkipAuth(t *testing.T) {
	srv := newTestHTTPServer(t)

	resp, err := srv.Client().Get(srv.URL + "/healthz")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok\n", string(body))

	resp, err = srv.Client().Get(srv.URL + "/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
