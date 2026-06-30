package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// requireAPIOnlyUser REFUSES to start unless FLEET_API_KEY belongs to an API-only
// Fleet user. API-only users have no UI session, carry their own audit identity,
// and — via Fleet's per-user role/team scoping — can be locked down to exactly
// the endpoints (and teams/fleets) the MCP needs, adding a Fleet-side
// authorization boundary on top of the MCP's own bearer auth. What each Fleet
// role can actually do (read vs create/run live queries) is documented in the
// README, not inferred here.
//
// Fails closed: if WhoAmI can't confirm the principal (Fleet unreachable or
// token invalid) we refuse to start rather than run with an unverified token —
// the MCP is non-functional without a reachable Fleet anyway.
func requireAPIOnlyUser(ctx context.Context, fleetClient *FleetClient) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	id, err := fleetClient.WhoAmI(ctx)
	if err != nil {
		logrus.Fatalf("could not verify FLEET_API_KEY via GET /api/v1/fleet/me (%v) — the MCP requires a reachable Fleet and an API-only token to start", err)
	}
	if !id.APIOnly {
		logrus.Fatalf("FLEET_API_KEY must belong to an API-only Fleet user, but %s is a UI user — refusing to start. Create one with `fleetctl user create --api-only` (no UI session, its own audit identity, scoped to only the endpoints/teams the MCP needs) and use its API token.", id.Email)
	}
	logrus.Infof("FLEET_API_KEY verified: API-only Fleet user %s", id.Email)
}

func main() {
	transport := flag.String("transport", "sse", "Transport protocol: 'sse' or 'stdio'")
	seed := flag.Bool("seed", false, "Seed Fleet with standard saved queries and exit")
	flag.Parse()

	// Root context cancelled on SIGINT/SIGTERM, so the startup checks and the
	// serving loop shut down gracefully (e.g. on a Render redeploy's SIGTERM)
	// rather than being hard-killed mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	if len(config.MCPAuthToken) < minMCPAuthTokenLen {
		logrus.Fatalf("MCP_AUTH_TOKEN is too weak (%d chars; need at least %d). Generate one with `openssl rand -hex 32`.", len(config.MCPAuthToken), minMCPAuthTokenLen)
	}

	// Stderr is required for stdio transport — logs must not corrupt the JSON-RPC stdout stream.
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(config.LogLevel)

	logrus.Info("starting Fleet MCP server")

	fleetClient := NewFleetClient(config.FleetBaseURL, config.FleetAPIKey, config.TLSSkipVerify, config.TLSCAFile)

	requireAPIOnlyUser(ctx, fleetClient)

	if *seed {
		SeedFleet(config, fleetClient)
		return
	}

	mcpServer := SetupMCPServer(config, fleetClient)

	if *transport == "stdio" {
		logrus.Info("transport: stdio")
		stdioServer := server.NewStdioServer(mcpServer)
		if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
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

	// /healthz is an unauthenticated liveness probe (see healthzHandler);
	// everything else goes through the middleware chain above.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.Handle("/", handler)

	// Explicit timeouts defeat Slowloris-style header/body starvation attacks
	// that pin connections to the server. ReadHeaderTimeout is the most
	// important — http.ListenAndServe leaves it as zero (unbounded). SSE
	// streams are long-lived so WriteTimeout/IdleTimeout are set generously
	// but bounded. ReadTimeout caps how long a slow client can take to send
	// the request body once the headers are in.
	httpServer := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE streams are long-lived; rely on idle/read timeouts
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("server error: %v", err)
		}
	}()

	// Block until SIGINT/SIGTERM, then drain in-flight requests gracefully.
	<-ctx.Done()
	logrus.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("graceful shutdown failed: %v", err)
	}
}
