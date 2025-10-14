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
						{"name": "get_weather"},
						{"name": "search_web"}
					]
				}
			}`,
			"prompts/list": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"prompts": [
						{"name": "code_review"},
						{"name": "summarize"}
					]
				}
			}`,
			"resources/list": `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"resources": [
						{"uri": "file:///data/doc1.txt"},
						{"uri": "file:///data/doc2.txt"}
					]
				}
			}`,
		},
		statusCode: 200,
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
	if rows[0]["has_prompts"] != "true" {
		t.Fatalf("expected has_prompts=true, got %s", rows[0]["has_prompts"])
	}
	if rows[0]["has_resources"] != "true" {
		t.Fatalf("expected has_resources=true, got %s", rows[0]["has_resources"])
	}
	if rows[0]["has_tools"] != "true" {
		t.Fatalf("expected has_tools=true, got %s", rows[0]["has_tools"])
	}
	if rows[0]["has_logging"] != "true" {
		t.Fatalf("expected has_logging=true, got %s", rows[0]["has_logging"])
	}
	if rows[0]["has_completions"] != "true" {
		t.Fatalf("expected has_completions=true, got %s", rows[0]["has_completions"])
	}
	if rows[0]["instructions"] != "Testing and demonstration server for MCP protocol features." {
		t.Fatalf("expected instructions=Testing and demonstration server for MCP protocol features., got %s", rows[0]["instructions"])
	}
	if rows[0]["tools"] != `["get_weather","search_web"]` {
		t.Fatalf("expected tools=[\"get_weather\",\"search_web\"], got %s", rows[0]["tools"])
	}
	if rows[0]["prompts"] != `["code_review","summarize"]` {
		t.Fatalf("expected prompts=[\"code_review\",\"summarize\"], got %s", rows[0]["prompts"])
	}
	if rows[0]["resources"] != `["file:///data/doc1.txt","file:///data/doc2.txt"]` {
		t.Fatalf("expected resources=[\"file:///data/doc1.txt\",\"file:///data/doc2.txt\"], got %s", rows[0]["resources"])
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
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
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
			"tools/list":     `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"test_tool"}]}}`,
			"prompts/list":   `{"jsonrpc":"2.0","id":1,"result":{"prompts":[{"name":"test_prompt"}]}}`,
			"resources/list": `{"jsonrpc":"2.0","id":1,"result":{"resources":[{"uri":"test://resource"}]}}`,
		},
		statusCode: 200,
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

	if rows[0]["protocol_version"] != "2025-03-26" {
		t.Fatalf("expected protocol_version=2025-03-26, got %s", rows[0]["protocol_version"])
	}
	if rows[0]["server_name"] != "example-servers/everything" {
		t.Fatalf("expected server_name=example-servers/everything, got %s", rows[0]["server_name"])
	}
}

// mockTransport is an HTTP transport that returns different responses based on the MCP method
type mockTransport struct {
	responses  map[string]string // method -> response
	statusCode int
	err        error
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
