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

func main() {
	transport := flag.String("transport", "sse", "Transport protocol: 'sse' or 'stdio'")
	seed := flag.Bool("seed", false, "Seed Fleet with standard saved queries and exit")
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
	// Per-IP token-bucket throttle: defends against bearer-token brute force
	// and burst floods that would otherwise amplify into Fleet API quota
	// exhaustion. Bucket size + refill rate are sized so normal MCP traffic
	// (a handful of tools/call requests per second) sails through, while a
	// flooder gets 429-throttled.
	rl := newIPRateLimiter(defaultPerIPRatePerSec, defaultPerIPBurst)
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
