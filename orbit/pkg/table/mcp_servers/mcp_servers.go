package mcp_servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	osqclient "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	// Socket address family constants (as strings for osquery comparison)
	afUnix  = "1"  // AF_UNIX - Unix domain sockets
	afInet  = "2"  // AF_INET - IPv4
	afInet6 = "10" // AF_INET6 - IPv6
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

// mcpTool represents a tool with its metadata
type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// mcpPrompt represents a prompt with its metadata
type mcpPrompt struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// mcpResource represents a resource with its metadata
type mcpResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

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
	Tools           []mcpTool     // List of tools with descriptions
	Prompts         []mcpPrompt   // List of prompts with descriptions
	Resources       []mcpResource // List of resources with descriptions
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

// mcpListResponse represents responses to tools/list, prompts/list, resources/list
type mcpListResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Tools     []mcpTool     `json:"tools"`
		Prompts   []mcpPrompt   `json:"prompts"`
		Resources []mcpResource `json:"resources"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// makeMCPRequest makes an MCP JSON-RPC request and returns the response body and session ID
func makeMCPRequest(ctx context.Context, url, method, sessionID string, params interface{}) ([]byte, string, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json,text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Extract session ID from response header
	responseSessionID := resp.Header.Get("Mcp-Session-Id")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
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

	return jsonData, responseSessionID, nil
}

// checkMCPServer attempts to connect to an MCP server at the given address and port.
// It returns information about the MCP server if it responds successfully, or nil otherwise.
func checkMCPServer(ctx context.Context, address, port string) *mcpServerInfo {
	// Build the URL - handle localhost and IPv6 addresses specially
	host := address

	// Check if the address is IPv6
	isIPv6 := false
	if ip := net.ParseIP(address); ip != nil {
		isIPv6 = ip.To4() == nil // If To4() returns nil, it's IPv6
	}

	// Handle wildcard and loopback addresses
	if address == "0.0.0.0" || address == "127.0.0.1" || address == "::" || address == "::1" || address == "" {
		host = "localhost"
	} else if isIPv6 {
		// IPv6 addresses must be wrapped in brackets in URLs
		host = "[" + address + "]"
	}

	url := fmt.Sprintf("http://%s:%s/mcp", host, port)

	// MCP initialize request
	// 2025-06-18 is the latest version of the MCP protocol as of 2025-10-14
	// TODO update this to the latest version of the MCP protocol when it is released
	initParams := map[string]interface{}{
		"protocolVersion": "2025-06-18",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
			"prompts":   map[string]interface{}{},
			"roots":     map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "fleetd",
			"version": "1.0.0",
		},
	}

	// Send initialize request without a session ID - server will provide one
	jsonData, sessionID, err := makeMCPRequest(ctx, url, "initialize", "", initParams)
	if err != nil {
		return nil
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
		// Initialize slices to empty so they marshal to [] instead of null
		Tools:     []mcpTool{},
		Prompts:   []mcpPrompt{},
		Resources: []mcpResource{},
	}

	// Fetch lists of tools, prompts, and resources if capabilities are available
	if info.HasTools {
		if listData, _, err := makeMCPRequest(ctx, url, "tools/list", sessionID, nil); err == nil {
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Str("port", port).Msg("tools/list returned error")
				} else {
					info.Tools = append(info.Tools, listResp.Result.Tools...)
				}
			}
		}
	}

	if info.HasPrompts {
		if listData, _, err := makeMCPRequest(ctx, url, "prompts/list", sessionID, nil); err == nil {
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Str("port", port).Msg("prompts/list returned error")
				} else {
					info.Prompts = append(info.Prompts, listResp.Result.Prompts...)
				}
			}
		}
	}

	if info.HasResources {
		if listData, _, err := makeMCPRequest(ctx, url, "resources/list", sessionID, nil); err == nil {
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Str("port", port).Msg("resources/list returned error")
				} else {
					info.Resources = append(info.Resources, listResp.Result.Resources...)
				}
			}
		}
	}

	return info
}

// Columns defines the schema for the mcp_servers table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("pid"),
		table.TextColumn("name"),
		table.TextColumn("cmdline"),
		table.IntegerColumn("port"),
		table.TextColumn("address"),
		table.TextColumn("protocol_version"),
		table.TextColumn("server_name"),
		table.TextColumn("server_title"),
		table.TextColumn("server_version"),
		table.IntegerColumn("has_logging"),
		table.IntegerColumn("has_completions"),
		table.TextColumn("instructions"),
		table.TextColumn("tools"),     // JSON array of tool names
		table.TextColumn("prompts"),   // JSON array of prompt names
		table.TextColumn("resources"), // JSON array of resource URIs
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
	sql := `
	SELECT DISTINCT lp.pid, lp.port, lp.address, lp.family, p.name, p.cmdline
	FROM listening_ports lp JOIN processes p ON lp.pid = p.pid
	`

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

		// Check family and log a warning if it's not a known address family
		if family, ok := row["family"]; ok {
			if family != afUnix && family != afInet && family != afInet6 {
				log.Warn().Str("family", family).Str("port", port).Str("pid", row["pid"]).Msg("unexpected family value")
			}
		}

		// Check if MCP server is active on this port
		mcpInfo := checkMCPServer(ctx, address, port)

		// Only include rows where an MCP server is actually responding
		if mcpInfo == nil {
			continue
		}

		// Convert lists to JSON
		toolsJSON, _ := json.Marshal(mcpInfo.Tools)
		promptsJSON, _ := json.Marshal(mcpInfo.Prompts)
		resourcesJSON, _ := json.Marshal(mcpInfo.Resources)

		// Create result row with all required columns
		result := map[string]string{
			"pid":              row["pid"],
			"name":             row["name"],
			"cmdline":          row["cmdline"],
			"port":             port,
			"address":          address,
			"protocol_version": mcpInfo.ProtocolVersion,
			"server_name":      mcpInfo.ServerName,
			"server_title":     mcpInfo.ServerTitle,
			"server_version":   mcpInfo.ServerVersion,
			"has_logging":      strconv.FormatBool(mcpInfo.HasLogging),
			"has_completions":  strconv.FormatBool(mcpInfo.HasCompletions),
			"instructions":     mcpInfo.Instructions,
			"tools":            string(toolsJSON),
			"prompts":          string(promptsJSON),
			"resources":        string(resourcesJSON),
		}
		results = append(results, result)
	}

	return results, nil
}
