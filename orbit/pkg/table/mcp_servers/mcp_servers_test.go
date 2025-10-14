package mcp_servers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	row  map[string]string
	rows []map[string]string
	err  error
}

func (m *mockClient) QueryRowContext(ctx context.Context, sql string) (map[string]string, error) {
	return m.row, m.err
}

func (m *mockClient) QueryRowsContext(ctx context.Context, sql string) ([]map[string]string, error) {
	return m.rows, m.err
}
func (m *mockClient) Close() {}

func TestGenerate_WithMCPServerActive(t *testing.T) {
	// Mock the osquery client
	oldClient := newClient
	defer func() { newClient = oldClient }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{
			{"pid": "1234", "port": "3001", "address": "127.0.0.1", "name": "node", "cmdline": "node mcp-server.js"},
		}}, nil
	}

	// Mock the HTTP client to return successful responses
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))

	mockClient.Transport = &mockTransport{
		responses: map[string]string{
			"initialize": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {
						"prompts": {},
						"resources": {"subscribe": true},
						"tools": {},
						"logging": {},
						"completions": {}
					},
					"serverInfo": {
						"name": "example-servers/everything",
						"title": "Everything Example Server",
						"version": "1.0.0"
					},
					"instructions": "Testing and demonstration server for MCP protocol features."
				}
			}`,
			"tools/list": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"tools": [
						{"name": "get_weather", "description": "Get weather for a location"},
						{"name": "search_web", "description": "Search the web"}
					]
				}
			}`,
			"prompts/list": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"prompts": [
						{"name": "code_review", "description": "Review code for quality"},
						{"name": "summarize", "description": "Summarize content"}
					]
				}
			}`,
			"resources/list": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"resources": [
						{"uri": "file:///data/doc1.txt", "name": "Document 1", "description": "First document"},
						{"uri": "file:///data/doc2.txt", "name": "Document 2", "description": "Second document"}
					]
				}
			}`,
		},
		statusCode: 200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "3001", rows[0]["port"])
	assert.Equal(t, "2025-03-26", rows[0]["protocol_version"])
	assert.Equal(t, "example-servers/everything", rows[0]["server_name"])
	assert.Equal(t, "Everything Example Server", rows[0]["server_title"])
	assert.Equal(t, "1.0.0", rows[0]["server_version"])
	assert.Equal(t, "true", rows[0]["has_logging"])
	assert.Equal(t, "true", rows[0]["has_completions"])
	assert.Equal(t, "Testing and demonstration server for MCP protocol features.", rows[0]["instructions"])
	assert.Equal(t, `[{"name":"get_weather","description":"Get weather for a location"},{"name":"search_web","description":"Search the web"}]`, rows[0]["tools"])
	assert.Equal(t, `[{"name":"code_review","description":"Review code for quality"},{"name":"summarize","description":"Summarize content"}]`, rows[0]["prompts"])
	assert.Equal(t, `[{"uri":"file:///data/doc1.txt","name":"Document 1","description":"First document"},{"uri":"file:///data/doc2.txt","name":"Document 2","description":"Second document"}]`, rows[0]["resources"])
}

func TestGenerate_WithMCPServerInactive(t *testing.T) {
	// Mock the osquery client
	oldClient := newClient
	defer func() { newClient = oldClient }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{
			{"pid": "5678", "port": "8080", "address": "0.0.0.0", "name": "nginx", "cmdline": "nginx"},
		}}, nil
	}

	// Mock the HTTP client to return an error (connection refused)
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))
	mockClient.Transport = &mockTransport{
		err: http.ErrServerClosed,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	require.NoError(t, err)
	// Should return 0 rows since no MCP server is active
	assert.Empty(t, rows)
}

func TestGenerate_MultipleActiveServers(t *testing.T) {
	// Mock the osquery client
	oldClient := newClient
	defer func() { newClient = oldClient }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{
			{"pid": "1234", "port": "3001", "address": "127.0.0.1", "name": "node", "cmdline": "node mcp1.js"},
			{"pid": "5678", "port": "3002", "address": "127.0.0.1", "name": "node", "cmdline": "node mcp2.js"},
		}}, nil
	}

	// Mock the HTTP client to return success for all
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))

	mockClient.Transport = &mockTransport{
		responses: map[string]string{
			"initialize": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {
						"prompts": {},
						"resources": {"subscribe": true},
						"tools": {},
						"logging": {},
						"completions": {}
					},
					"serverInfo": {
						"name": "example-servers/everything",
						"title": "Everything Example Server",
						"version": "1.0.0"
					},
					"instructions": "Testing and demonstration server for MCP protocol features."
				}
			}`,
			"tools/list":     `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
			"prompts/list":   `{"jsonrpc":"2.0","id":1,"result":{"prompts":[]}}`,
			"resources/list": `{"jsonrpc":"2.0","id":1,"result":{"resources":[]}}`,
		},
		statusCode: 200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestGenerate_WithSSEResponse(t *testing.T) {
	// Mock the osquery client
	oldClient := newClient
	defer func() { newClient = oldClient }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{
			{"pid": "1234", "port": "3001", "address": "127.0.0.1", "name": "node", "cmdline": "node mcp-server.js"},
		}}, nil
	}

	// Mock the HTTP client to return SSE-formatted responses
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))

	// Full SSE format with event, id, and data lines
	mockClient.Transport = &mockTransport{
		responses: map[string]string{
			"initialize": `event: message
id: 4b74868e-307e-416c-951d-6f305856cb43_1760466849580_vmzdy66v
data: {"result":{"protocolVersion":"2025-03-26","capabilities":{"prompts":{},"resources":{"subscribe":true},"tools":{},"logging":{},"completions":{}},"serverInfo":{"name":"example-servers/everything","title":"Everything Example Server","version":"1.0.0"},"instructions":"Testing and demonstration server for MCP protocol features."},"jsonrpc":"2.0","id":1}
`,
			"tools/list":     `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"test_tool","description":"A test tool"}]}}`,
			"prompts/list":   `{"jsonrpc":"2.0","id":1,"result":{"prompts":[{"name":"test_prompt","description":"A test prompt"}]}}`,
			"resources/list": `{"jsonrpc":"2.0","id":1,"result":{"resources":[{"uri":"test://resource","name":"Test Resource","description":"A test resource"}]}}`,
		},
		statusCode: 200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "2025-03-26", rows[0]["protocol_version"])
	assert.Equal(t, "example-servers/everything", rows[0]["server_name"])
}

func TestGenerate_WithIPv6Address(t *testing.T) {
	// Mock the osquery client with IPv6 addresses
	oldClient := newClient
	defer func() { newClient = oldClient }()
	newClient = func(socket string, timeout time.Duration) (osqClient, error) {
		return &mockClient{rows: []map[string]string{
			{"pid": "1234", "port": "3001", "address": "::1", "family": afInet6, "name": "node", "cmdline": "node mcp-server.js"},
			{"pid": "5678", "port": "3002", "address": "2001:db8::1", "family": afInet6, "name": "node", "cmdline": "node mcp-server2.js"},
			{"pid": "9999", "port": "3003", "address": "::", "family": afInet6, "name": "node", "cmdline": "node mcp-server3.js"},
		}}, nil
	}

	// Mock the HTTP client to return successful responses
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))

	// Track which URLs were actually requested to verify IPv6 bracket handling
	requestedURLs := []string{}
	mockClient.Transport = &mockTransportWithURLTracking{
		requestedURLs: &requestedURLs,
		responses: map[string]string{
			"initialize": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {
						"tools": {}
					},
					"serverInfo": {
						"name": "test-server",
						"title": "Test Server",
						"version": "1.0.0"
					}
				}
			}`,
			"tools/list": `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
		},
		statusCode: 200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	// Verify that IPv6 loopback (::1) was converted to localhost
	assert.Contains(t, requestedURLs, "http://localhost:3001/mcp")

	// Verify that IPv6 wildcard (::) was converted to localhost
	assert.Contains(t, requestedURLs, "http://localhost:3003/mcp")

	// Verify that regular IPv6 addresses are wrapped in brackets
	assert.Contains(t, requestedURLs, "http://[2001:db8::1]:3002/mcp")

	// All should return the same server info
	assert.Equal(t, "test-server", rows[0]["server_name"])
	assert.Equal(t, "test-server", rows[1]["server_name"])
	assert.Equal(t, "test-server", rows[2]["server_name"])
}

// mockTransport is an HTTP transport that returns different responses based on the MCP method
type mockTransport struct {
	responses  map[string]string // method -> response
	statusCode int
	err        error
}

// mockTransportWithURLTracking is like mockTransport but also tracks requested URLs
type mockTransportWithURLTracking struct {
	requestedURLs *[]string
	responses     map[string]string
	statusCode    int
	err           error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Parse the request to determine which method is being called
	var reqBody map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)
	_ = json.Unmarshal(bodyBytes, &reqBody)

	method, _ := reqBody["method"].(string)
	responseBody := m.responses[method]
	if responseBody == "" {
		responseBody = `{"jsonrpc":"2.0","id":1,"result":{}}`
	}

	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
	}, nil
}

func (m *mockTransportWithURLTracking) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Track the requested URL
	*m.requestedURLs = append(*m.requestedURLs, req.URL.String())

	// Parse the request to determine which method is being called
	var reqBody map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)
	_ = json.Unmarshal(bodyBytes, &reqBody)

	method, _ := reqBody["method"].(string)
	responseBody := m.responses[method]
	if responseBody == "" {
		responseBody = `{"jsonrpc":"2.0","id":1,"result":{}}`
	}

	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		Header:     http.Header{"Mcp-Session-Id": []string{"test-session-id"}},
	}, nil
}
