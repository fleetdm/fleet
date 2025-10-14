package mcp_servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// checkMCPServer attempts to connect to an MCP server at the given address and port.
// It returns 1 if the server responds successfully to an MCP initialize request, 0 otherwise.
func checkMCPServer(ctx context.Context, address, port string) string {
	// Build the URL - handle localhost specially
	host := address
	if address == "0.0.0.0" || address == "127.0.0.1" || address == "" {
		host = "localhost"
	}
	url := fmt.Sprintf("http://%s:%s/mcp", host, port)

	// Build the MCP initialize request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
				"prompts":   map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "fleet-osquery",
				"version": "1.0.0",
			},
		},
	}

	body, err := json.Marshal(initRequest)
	if err != nil {
		return "0"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "0"
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json,text/event-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "0"
	}
	defer resp.Body.Close()

	// Check if we got a successful response (2xx status code)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "1"
	}

	return "0"
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
		mcpActive := checkMCPServer(ctx, address, port)

		// Only include rows where an MCP server is actually responding
		if mcpActive != "1" {
			continue
		}

		// Create result row with all required columns
		result := map[string]string{
			"pid":        row["pid"],
			"name":       row["name"],
			"cmdline":    row["cmdline"],
			"port":       port,
			"protocol":   row["protocol"],
			"address":    address,
			"mcp_active": mcpActive,
		}
		results = append(results, result)
	}

	return results, nil
}
