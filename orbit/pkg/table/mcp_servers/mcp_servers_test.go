package mcp_servers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/osquery/osquery-go/plugin/table"
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

	// Mock the HTTP client to return a successful response
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))
	mockResponse := `{
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
	}`
	mockClient.Transport = &mockTransport{
		responseBody: mockResponse,
		statusCode:   200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0]["mcp_active"] != "1" {
		t.Fatalf("expected mcp_active=1, got %s", rows[0]["mcp_active"])
	}
	if rows[0]["port"] != "3001" {
		t.Fatalf("expected port=3001, got %s", rows[0]["port"])
	}
	if rows[0]["protocol_version"] != "2025-03-26" {
		t.Fatalf("expected protocol_version=2025-03-26, got %s", rows[0]["protocol_version"])
	}
	if rows[0]["server_name"] != "example-servers/everything" {
		t.Fatalf("expected server_name=example-servers/everything, got %s", rows[0]["server_name"])
	}
	if rows[0]["server_title"] != "Everything Example Server" {
		t.Fatalf("expected server_title=Everything Example Server, got %s", rows[0]["server_title"])
	}
	if rows[0]["server_version"] != "1.0.0" {
		t.Fatalf("expected server_version=1.0.0, got %s", rows[0]["server_version"])
	}
	if rows[0]["has_prompts"] != "1" {
		t.Fatalf("expected has_prompts=1, got %s", rows[0]["has_prompts"])
	}
	if rows[0]["has_resources"] != "1" {
		t.Fatalf("expected has_resources=1, got %s", rows[0]["has_resources"])
	}
	if rows[0]["has_tools"] != "1" {
		t.Fatalf("expected has_tools=1, got %s", rows[0]["has_tools"])
	}
	if rows[0]["has_logging"] != "1" {
		t.Fatalf("expected has_logging=1, got %s", rows[0]["has_logging"])
	}
	if rows[0]["has_completions"] != "1" {
		t.Fatalf("expected has_completions=1, got %s", rows[0]["has_completions"])
	}
	if rows[0]["instructions"] != "Testing and demonstration server for MCP protocol features." {
		t.Fatalf("expected instructions=Testing and demonstration server for MCP protocol features., got %s", rows[0]["instructions"])
	}
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
	if err != nil {
		t.Fatal(err)
	}
	// Should return 0 rows since no MCP server is active
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
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
	mockResponse := `{
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
	}`
	mockClient.Transport = &mockTransport{
		responseBody: mockResponse,
		statusCode:   200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["mcp_active"] != "1" {
		t.Fatalf("expected mcp_active=1 for first row, got %s", rows[0]["mcp_active"])
	}
	if rows[1]["mcp_active"] != "1" {
		t.Fatalf("expected mcp_active=1 for second row, got %s", rows[1]["mcp_active"])
	}
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

	// Mock the HTTP client to return an SSE-formatted response
	oldHTTPClient := httpClient
	defer func() { httpClient = oldHTTPClient }()
	mockClient := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))
	// Full SSE format with event, id, and data lines
	sseResponse := `event: message
id: 4b74868e-307e-416c-951d-6f305856cb43_1760466849580_vmzdy66v
data: {"result":{"protocolVersion":"2025-03-26","capabilities":{"prompts":{},"resources":{"subscribe":true},"tools":{},"logging":{},"completions":{}},"serverInfo":{"name":"example-servers/everything","title":"Everything Example Server","version":"1.0.0"},"instructions":"Testing and demonstration server for MCP protocol features."},"jsonrpc":"2.0","id":1}
`
	mockClient.Transport = &mockTransport{
		responseBody: sseResponse,
		statusCode:   200,
	}
	httpClient = mockClient

	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}

	rows, err := Generate(context.Background(), qc, "/tmp/osq")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0]["mcp_active"] != "1" {
		t.Fatalf("expected mcp_active=1, got %s", rows[0]["mcp_active"])
	}
	if rows[0]["protocol_version"] != "2025-03-26" {
		t.Fatalf("expected protocol_version=2025-03-26, got %s", rows[0]["protocol_version"])
	}
	if rows[0]["server_name"] != "example-servers/everything" {
		t.Fatalf("expected server_name=example-servers/everything, got %s", rows[0]["server_name"])
	}
}

// mockTransport is a simple HTTP transport that returns a fixed response or error
type mockTransport struct {
	responseBody string
	statusCode   int
	err          error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.responseBody)),
	}, nil
}
