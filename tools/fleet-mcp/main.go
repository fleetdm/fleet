package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func main() {
	transport := flag.String("transport", "sse", "Transport protocol: 'sse' or 'stdio'")
	flag.Parse()

	config := LoadConfig()

	if strings.TrimSpace(config.FleetBaseURL) == "" {
		logrus.Fatalf("FLEET_BASE_URL is required but is not set")
	}
	if strings.TrimSpace(config.FleetAPIKey) == "" {
		logrus.Fatalf("FLEET_API_KEY is required but is not set")
	}
	if strings.TrimSpace(config.MCPAuthToken) == "" {
		logrus.Fatalf("MCP_AUTH_TOKEN is required at startup for all transports, including stdio, but is not set")
	}

	// Stderr is required for stdio transport — logs must not corrupt the JSON-RPC stdout stream.
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(config.LogLevel)

	logrus.Info("starting Fleet MCP server")

	fleetClient := NewFleetClient(config.FleetBaseURL, config.FleetAPIKey, config.TLSSkipVerify, config.TLSCAFile)
	mcpServer := SetupMCPServer(config, fleetClient)

	if *transport == "stdio" {
		logrus.Info("transport: stdio")
		stdioServer := server.NewStdioServer(mcpServer)
		if err := stdioServer.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			logrus.Fatalf("server error: %v", err)
		}
		return
	}

	logrus.Infof("transport: SSE — listening on :%s", config.Port)
	sseServer := server.NewSSEServer(mcpServer)
	var handler http.Handler = sseServer
	logrus.Info("authentication enabled")
	handler = bearerAuthMiddleware(config.MCPAuthToken, handler)
	handler = mcpRouteGuard(handler)
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logrus.Fatalf("server error: %v", err)
	}
}
