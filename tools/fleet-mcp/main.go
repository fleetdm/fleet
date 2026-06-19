package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// maxRequestBodyBytes bounds the body of any incoming MCP/SSE request. JSON-RPC
// payloads are tiny (kilobytes); 1 MiB is a generous ceiling that defeats
// memory-exhaustion attacks via oversized POST bodies.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// minMCPAuthTokenLen is the minimum accepted length for MCP_AUTH_TOKEN. A
// high-entropy token is the real brute-force defense; 32 chars is trivially met
// by the documented `openssl rand -hex 32` (64 chars).
const minMCPAuthTokenLen = 32

// limitBodyMiddleware caps r.Body so handlers downstream cannot accidentally
// buffer arbitrarily large payloads from a hostile client.
func limitBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// checkTokenPosture logs the Fleet principal behind FLEET_API_KEY and the
// resulting MCP "mode" (read-only vs write-capable), and warns when the token
// is over-privileged or not an API-only user.
// The MCP's capability mirrors the token's Fleet role.
func checkTokenPosture(fleetClient *FleetClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := fleetClient.WhoAmI(ctx)
	if err != nil {
		logrus.Debugf("skipping Fleet token posture check (could not reach /api/v1/fleet/me): %v", err)
		return
	}

	// Resolve the token's privilege from its global role OR its per-fleet roles.
	roleDesc, writeCapable := id.privilege()
	mode := "read-only"
	if writeCapable {
		mode = "write-capable (can create/run live queries on hosts)"
	}
	logrus.Infof("Fleet token identity: %s (%s, api_only=%t) — %s mode", id.Email, roleDesc, id.APIOnly, mode)

	if !id.APIOnly {
		logrus.Warn("FLEET_API_KEY is a UI user, not an API-only user — prefer a dedicated API-only Fleet user (no UI access, its own audit identity, long-lived token)")
	}
	if writeCapable {
		logrus.Warnf("FLEET_API_KEY is write-capable (%s): the MCP can create queries and run live queries (arbitrary osquery) on hosts it can reach. Use an observer-only token to run the MCP read-only.", roleDesc)
	}
}

func main() {
	transport := flag.String("transport", "sse", "Transport protocol: 'sse' or 'stdio'")
	seed := flag.Bool("seed", false, "Seed Fleet with standard saved queries and exit")
	flag.Parse()

	config := LoadConfig()

	if strings.TrimSpace(config.FleetBaseURL) == "" {
		logrus.Fatalf("FLEET_BASE_URL is required but is not set")
	}
	if err := validateFleetBaseURL(config.FleetBaseURL); err != nil {
		logrus.Fatalf("%v", err)
	}
	if strings.TrimSpace(config.FleetAPIKey) == "" {
		logrus.Fatalf("FLEET_API_KEY is required but is not set")
	}
	if strings.TrimSpace(config.MCPAuthToken) == "" {
		logrus.Fatalf("MCP_AUTH_TOKEN is required at startup for all transports, including stdio, but is not set")
	}
	if len(config.MCPAuthToken) < minMCPAuthTokenLen {
		logrus.Fatalf("MCP_AUTH_TOKEN is too weak (%d chars; need at least %d). Generate one with `openssl rand -hex 32`.", len(config.MCPAuthToken), minMCPAuthTokenLen)
	}

	// Stderr is required for stdio transport — logs must not corrupt the JSON-RPC stdout stream.
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(config.LogLevel)

	logrus.Info("starting Fleet MCP server")

	fleetClient := NewFleetClient(config.FleetBaseURL, config.FleetAPIKey, config.TLSSkipVerify, config.TLSCAFile)

	// Surface the Fleet token's privilege at startup.
	checkTokenPosture(fleetClient)

	// Best-effort cleanup of any fleet-mcp-temp-* saved queries left over from
	// previous runs whose DELETE failed. Synchronous so any temporary cleanup
	// failures show up immediately in startup logs; errors are logged not fatal.
	fleetClient.SweepLeftoverTempQueries(context.Background())

	if *seed {
		SeedFleet(config, fleetClient)
		return
	}

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
	handler = limitBodyMiddleware(handler)
	// Global token-bucket throttle: a single shared bucket bounds burst floods /
	// request amplification without per-client IP attribution (the MCP is
	// single-tenant, so there's no per-client fairness to provide, and keying on
	// X-Forwarded-For behind a proxy is a footgun). Brute force of MCP_AUTH_TOKEN
	// is handled by the token-strength check above, not by rate.
	rl := newGlobalRateLimiter(defaultGlobalRatePerSec, defaultGlobalBurst)
	handler = rl.Middleware(handler)
	// Explicit timeouts defeat Slowloris-style header/body starvation attacks
	// that pin connections to the server. ReadHeaderTimeout is the most
	// important — http.ListenAndServe leaves it as zero (unbounded). SSE
	// streams are long-lived so WriteTimeout/IdleTimeout are set generously
	// but bounded. ReadTimeout caps how long a slow client can take to send
	// the request body once the headers are in.
	httpServer := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE streams are long-lived; rely on idle/read timeouts
		IdleTimeout:       120 * time.Second,
	}
	if err := httpServer.ListenAndServe(); err != nil {
		logrus.Fatalf("server error: %v", err)
	}
}
