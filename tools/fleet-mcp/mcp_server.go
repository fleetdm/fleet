package main

import (
	"github.com/mark3labs/mcp-go/server"
)

const defaultEndpointsPerPage = 50

// SetupMCPServer creates and configures the MCP server with all available tools.
// Tool registrations are split by domain across mcp_tools_*.go files.
func SetupMCPServer(config *Config, fleetClient *FleetClient) *server.MCPServer {
	s := server.NewMCPServer("fleet-mcp", "1.0.0", server.WithLogging())

	registerHostTools(s, fleetClient)
	registerQueryTools(s, fleetClient)
	registerPolicyTools(s, fleetClient)

	return s
}
