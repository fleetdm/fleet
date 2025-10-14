package mcp_servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/uuid"
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
	Tools           []string // List of tool names
	Prompts         []string // List of prompt names
	Resources       []string // List of resource URIs
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
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
		Prompts []struct {
			Name string `json:"name"`
		} `json:"prompts"`
		Resources []struct {
			URI string `json:"uri"`
		} `json:"resources"`
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
		log.Debug().Err(err).Str("method", method).Msg("failed to marshal MCP request")
		return nil, "", err
	}

	log.Debug().Str("url", url).Str("method", method).Str("session_id", sessionID).Str("request", string(bodyJSON)).Msg("sending MCP request")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		log.Debug().Err(err).Str("method", method).Msg("failed to create HTTP request")
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json,text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Str("url", url).Str("method", method).Msg("HTTP request failed")
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Debug().Int("status", resp.StatusCode).Str("method", method).Msg("MCP server returned non-2xx status")
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Extract session ID from response header
	responseSessionID := resp.Header.Get("Mcp-Session-Id")
	if responseSessionID != "" {
		log.Debug().Str("method", method).Str("session_id", responseSessionID).Msg("received session ID from server")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debug().Err(err).Str("method", method).Msg("failed to read response body")
		return nil, "", err
	}

	log.Debug().Str("method", method).Str("raw_response", string(bodyBytes)).Msg("received MCP response")

	// Handle SSE format (Server-Sent Events) - responses may contain lines like:
	// event: message
	// id: ...
	// data: {...}
	jsonData := bodyBytes

	// Check if this looks like SSE format (contains "data: " on a line)
	if bytes.Contains(bodyBytes, []byte("data: ")) {
		log.Debug().Str("method", method).Msg("detected SSE format, extracting JSON")
		// Extract JSON from SSE format
		lines := bytes.Split(bodyBytes, []byte("\n"))
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("data: ")) {
				jsonData = bytes.TrimPrefix(line, []byte("data: "))
				log.Debug().Str("method", method).Str("extracted_json", string(jsonData)).Msg("extracted JSON from SSE")
				break
			}
		}
	}

	return jsonData, responseSessionID, nil
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

	log.Debug().Str("url", url).Msg("connecting to MCP server")

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

	if sessionID == "" {
		log.Debug().Str("url", url).Msg("server did not provide session ID")
		// Generate one ourselves if server doesn't provide it
		sessionID = uuid.New().String()
		log.Debug().Str("url", url).Str("session_id", sessionID).Msg("generated client-side session ID")
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
		Tools:     []string{},
		Prompts:   []string{},
		Resources: []string{},
	}

	// Fetch lists of tools, prompts, and resources if capabilities are available
	if info.HasTools {
		log.Debug().Str("url", url).Str("session_id", sessionID).Msg("fetching tools list")
		if listData, _, err := makeMCPRequest(ctx, url, "tools/list", sessionID, nil); err == nil {
			log.Debug().Str("tools_list_data", string(listData)).Msg("received tools/list data")
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Msg("tools/list returned error")
				} else {
					log.Debug().Int("tools_in_response", len(listResp.Result.Tools)).Msg("unmarshaled tools/list response")
					for _, tool := range listResp.Result.Tools {
						log.Debug().Str("tool_name", tool.Name).Msg("adding tool")
						info.Tools = append(info.Tools, tool.Name)
					}
					log.Debug().Int("count", len(info.Tools)).Strs("tools", info.Tools).Msg("parsed tools list")
				}
			} else {
				log.Debug().Err(err).Str("data", string(listData)).Msg("failed to unmarshal tools/list response")
			}
		} else {
			log.Debug().Err(err).Msg("failed to fetch tools list")
		}
	} else {
		log.Debug().Msg("HasTools is false, skipping tools list")
	}

	if info.HasPrompts {
		log.Debug().Str("url", url).Str("session_id", sessionID).Msg("fetching prompts list")
		if listData, _, err := makeMCPRequest(ctx, url, "prompts/list", sessionID, nil); err == nil {
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Msg("prompts/list returned error")
				} else {
					for _, prompt := range listResp.Result.Prompts {
						info.Prompts = append(info.Prompts, prompt.Name)
					}
					log.Debug().Int("count", len(info.Prompts)).Strs("prompts", info.Prompts).Msg("parsed prompts list")
				}
			} else {
				log.Debug().Err(err).Str("data", string(listData)).Msg("failed to unmarshal prompts/list response")
			}
		} else {
			log.Debug().Err(err).Msg("failed to fetch prompts list")
		}
	}

	if info.HasResources {
		log.Debug().Str("url", url).Str("session_id", sessionID).Msg("fetching resources list")
		if listData, _, err := makeMCPRequest(ctx, url, "resources/list", sessionID, nil); err == nil {
			var listResp mcpListResponse
			if err := json.Unmarshal(listData, &listResp); err == nil {
				// Check if the response contains an error
				if listResp.Error != nil {
					log.Warn().Int("code", listResp.Error.Code).Str("message", listResp.Error.Message).Msg("resources/list returned error")
				} else {
					for _, resource := range listResp.Result.Resources {
						info.Resources = append(info.Resources, resource.URI)
					}
					log.Debug().Int("count", len(info.Resources)).Strs("resources", info.Resources).Msg("parsed resources list")
				}
			} else {
				log.Debug().Err(err).Str("data", string(listData)).Msg("failed to unmarshal resources/list response")
			}
		} else {
			log.Debug().Err(err).Msg("failed to fetch resources list")
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
		table.IntegerColumn("has_prompts"),
		table.IntegerColumn("has_resources"),
		table.IntegerColumn("has_tools"),
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

		log.Debug().
			Str("port", port).
			Str("server_name", mcpInfo.ServerName).
			Int("tools_count", len(mcpInfo.Tools)).
			Int("prompts_count", len(mcpInfo.Prompts)).
			Int("resources_count", len(mcpInfo.Resources)).
			Str("tools_json", string(toolsJSON)).
			Str("prompts_json", string(promptsJSON)).
			Str("resources_json", string(resourcesJSON)).
			Msg("creating result row for MCP server")

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
			"has_prompts":      strconv.FormatBool(mcpInfo.HasPrompts),
			"has_resources":    strconv.FormatBool(mcpInfo.HasResources),
			"has_tools":        strconv.FormatBool(mcpInfo.HasTools),
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
