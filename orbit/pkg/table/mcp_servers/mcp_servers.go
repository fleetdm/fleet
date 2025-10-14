package mcp_servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	osqclient "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

// osqClient abstracts the osquery client for ease of testing.
type osqClient interface {
	QueryRowContext(ctx context.Context, sql string) (map[string]string, error)
	QueryRowsContext(ctx context.Context, sql string) ([]map[string]string, error)
	Close()
}

var newClient = func(socket string, timeout time.Duration) (osqClient, error) {
	return osqclient.NewClient(socket, timeout)
}

// httpClient is a variable to allow mocking in tests
var httpClient = fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Second))

// mcpServerInfo holds information about an MCP server
type mcpServerInfo struct {
	ProtocolVersion string
	ServerName      string
	ServerTitle     string
	ServerVersion   string
	HasPrompts      bool
	HasResources    bool
	HasTools        bool
	HasLogging      bool
	HasCompletions  bool
	Instructions    string
}

// mcpResponse represents the JSON-RPC response from an MCP server
type mcpResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Prompts     *json.RawMessage `json:"prompts"`
			Resources   *json.RawMessage `json:"resources"`
			Tools       *json.RawMessage `json:"tools"`
			Logging     *json.RawMessage `json:"logging"`
			Completions *json.RawMessage `json:"completions"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Instructions string `json:"instructions"`
	} `json:"result"`
}

// checkMCPServer attempts to connect to an MCP server at the given address and port.
// It returns information about the MCP server if it responds successfully, or nil otherwise.
func checkMCPServer(ctx context.Context, address, port string) *mcpServerInfo {
	// Build the URL - handle localhost specially
	host := address
	if address == "0.0.0.0" || address == "127.0.0.1" || address == "" {
		host = "localhost"
	}
	url := fmt.Sprintf("http://%s:%s/mcp", host, port)

	// MCP initialize request body
	// 2025-06-18 is the latest version of the MCP protocol as of 2025-10-14
	// TODO update this to the latest version of the MCP protocol when it is released
	body := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2025-06-18",
			"capabilities": {
				"tools": {},
				"resources": {},
				"prompts": {}
			},
			"clientInfo": {
				"name": "fleetd",
				"version": "1.0.0"
			}
		}
	}`

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(body))
	if err != nil {
		return nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json,text/event-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Check if we got a successful response (2xx status code)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	// Read and parse the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Handle SSE format (Server-Sent Events) - responses may contain lines like:
	// event: message
	// id: ...
	// data: {...}
	jsonData := bodyBytes

	// Check if this looks like SSE format (contains "data: " on a line)
	if bytes.Contains(bodyBytes, []byte("data: ")) {
		// Extract JSON from SSE format
		lines := bytes.Split(bodyBytes, []byte("\n"))
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("data: ")) {
				jsonData = bytes.TrimPrefix(line, []byte("data: "))
				break
			}
		}
	}

	var mcpResp mcpResponse
	if err := json.Unmarshal(jsonData, &mcpResp); err != nil {
		// If JSON parsing fails, return nil so the row is not included
		// (we only want rows where we can successfully identify an MCP server)
		return nil
	}

	// Build the server info struct
	info := &mcpServerInfo{
		ProtocolVersion: mcpResp.Result.ProtocolVersion,
		ServerName:      mcpResp.Result.ServerInfo.Name,
		ServerTitle:     mcpResp.Result.ServerInfo.Title,
		ServerVersion:   mcpResp.Result.ServerInfo.Version,
		HasPrompts:      mcpResp.Result.Capabilities.Prompts != nil,
		HasResources:    mcpResp.Result.Capabilities.Resources != nil,
		HasTools:        mcpResp.Result.Capabilities.Tools != nil,
		HasLogging:      mcpResp.Result.Capabilities.Logging != nil,
		HasCompletions:  mcpResp.Result.Capabilities.Completions != nil,
		Instructions:    mcpResp.Result.Instructions,
	}

	return info
}

// Columns defines the schema for the mcp_servers table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("pid"),
		table.TextColumn("name"),
		table.TextColumn("cmdline"),
		table.TextColumn("port"),
		table.TextColumn("protocol"),
		table.TextColumn("address"),
		table.IntegerColumn("mcp_active"),
		table.TextColumn("protocol_version"),
		table.TextColumn("server_name"),
		table.TextColumn("server_title"),
		table.TextColumn("server_version"),
		table.IntegerColumn("has_prompts"),
		table.IntegerColumn("has_resources"),
		table.IntegerColumn("has_tools"),
		table.IntegerColumn("has_logging"),
		table.IntegerColumn("has_completions"),
		table.TextColumn("instructions"),
	}
}

// Generate connects to the running osqueryd over the provided socket and queries
// the listening_ports and processes tables to find processes with listening ports,
// then checks each port to see if an MCP server is responding.
func Generate(ctx context.Context, queryContext table.QueryContext, socket string) ([]map[string]string, error) {
	// Ensure we don't hang forever if osquery is unresponsive.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Open an osquery client using the extension socket.
	c, err := newClient(socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("open osquery client: %w", err)
	}
	defer c.Close()

	// Get the running processes with listening ports
	sql := `SELECT DISTINCT lp.pid, lp.port, lp.protocol, lp.address, p.name, p.cmdline FROM listening_ports lp JOIN processes p ON lp.pid = p.pid WHERE lp.protocol IN ('6', '17')`

	rows, err := c.QueryRowsContext(ctx, sql)
	if err != nil {
		return nil, err
	}

	// For each row, check if there's an MCP server listening on that port
	results := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		port, ok := row["port"]
		if !ok || port == "" {
			continue
		}

		address, ok := row["address"]
		if !ok {
			address = "127.0.0.1"
		}

		// Check if MCP server is active on this port
		mcpInfo := checkMCPServer(ctx, address, port)

		// Only include rows where an MCP server is actually responding
		if mcpInfo == nil {
			continue
		}

		// Convert booleans to "0" or "1" for osquery compatibility
		boolToStr := func(b bool) string {
			if b {
				return "1"
			}
			return "0"
		}

		// Create result row with all required columns
		result := map[string]string{
			"pid":              row["pid"],
			"name":             row["name"],
			"cmdline":          row["cmdline"],
			"port":             port,
			"protocol":         row["protocol"],
			"address":          address,
			"mcp_active":       "1",
			"protocol_version": mcpInfo.ProtocolVersion,
			"server_name":      mcpInfo.ServerName,
			"server_title":     mcpInfo.ServerTitle,
			"server_version":   mcpInfo.ServerVersion,
			"has_prompts":      boolToStr(mcpInfo.HasPrompts),
			"has_resources":    boolToStr(mcpInfo.HasResources),
			"has_tools":        boolToStr(mcpInfo.HasTools),
			"has_logging":      boolToStr(mcpInfo.HasLogging),
			"has_completions":  boolToStr(mcpInfo.HasCompletions),
			"instructions":     mcpInfo.Instructions,
		}
		results = append(results, result)
	}

	return results, nil
}
