package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleetmcp"
	"github.com/mark3labs/mcp-go/server"
)

// minMCPAuthTokenLen is the minimum accepted length for MCP_AUTH_TOKEN. A
// high-entropy token is the real brute-force defense; 32 chars is trivially met
// by the documented `openssl rand -hex 32` (64 chars).
const minMCPAuthTokenLen = 32

func fatal(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, args...)
	os.Exit(1)
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
func requireAPIOnlyUser(ctx context.Context, logger *slog.Logger, fleetClient *fleetmcp.FleetClient) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	id, err := fleetClient.WhoAmI(ctx)
	if err != nil {
		fatal(ctx, logger, "could not verify FLEET_API_KEY via GET /api/v1/fleet/me — the MCP requires a reachable Fleet and an API-only token to start", "err", err)
	}
	if !id.APIOnly {
		fatal(ctx, logger, "FLEET_API_KEY must belong to an API-only Fleet user, but it belongs to a UI user — refusing to start. Create one with `fleetctl user create --api-only` (no UI session, its own audit identity, scoped to only the endpoints/teams the MCP needs) and use its API token.", "email", id.Email)
	}
	logger.InfoContext(ctx, "FLEET_API_KEY verified: API-only Fleet user", "email", id.Email)
}

func main() {
	transport := flag.String("transport", "http", "Transport protocol: 'http' (Streamable HTTP at /mcp plus legacy SSE at /sse) or 'stdio'. 'sse' is accepted as an alias of 'http'.")
	seed := flag.Bool("seed", false, "Seed Fleet with standard saved queries and exit")
	flag.Parse()

	// Root context cancelled on SIGINT/SIGTERM, so the startup checks and the
	// serving loop shut down gracefully (e.g. on a Render redeploy's SIGTERM)
	// rather than being hard-killed mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := LoadConfig()
	if err != nil {
		fatal(ctx, slog.New(slog.NewTextHandler(os.Stderr, nil)), "invalid configuration", "err", err)
	}

	// Stderr is required for stdio transport — logs must not corrupt the JSON-RPC stdout stream.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: config.LogLevel}))

	switch *transport {
	case "http", "sse", "stdio":
	default:
		fatal(ctx, logger, "unknown -transport: use 'http' or 'stdio'", "transport", *transport)
	}

	if strings.TrimSpace(config.FleetBaseURL) == "" {
		fatal(ctx, logger, "FLEET_BASE_URL is required but is not set")
	}
	if strings.TrimSpace(config.FleetAPIKey) == "" {
		fatal(ctx, logger, "FLEET_API_KEY is required but is not set")
	}
	if strings.TrimSpace(config.MCPAuthToken) == "" {
		fatal(ctx, logger, "MCP_AUTH_TOKEN is required at startup for all transports, including stdio, but is not set")
	}
	if len(config.MCPAuthToken) < minMCPAuthTokenLen {
		fatal(ctx, logger, "MCP_AUTH_TOKEN is too weak. Generate one with `openssl rand -hex 32`.", "length", len(config.MCPAuthToken), "min_length", minMCPAuthTokenLen)
	}

	logger.InfoContext(ctx, "starting Fleet MCP server")

	fleetClient, err := fleetmcp.NewFleetClient(ctx, config.FleetBaseURL, config.FleetAPIKey, config.TLSSkipVerify, config.TLSCAFile, logger)
	if err != nil {
		fatal(ctx, logger, "invalid Fleet connection settings", "err", err)
	}

	requireAPIOnlyUser(ctx, logger, fleetClient)

	if *seed {
		fleetmcp.SeedFleet(ctx, fleetClient)
		return
	}

	mcpServer := fleetmcp.SetupMCPServer(ctx, fleetClient)

	if *transport == "stdio" {
		logger.InfoContext(ctx, "transport: stdio")
		stdioServer := server.NewStdioServer(mcpServer)
		if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
			fatal(ctx, logger, "server error", "err", err)
		}
		return
	}

	logger.InfoContext(ctx, "transport: HTTP (Streamable HTTP at /mcp, SSE at /sse)", "port", config.Port)
	logger.InfoContext(ctx, "authentication enabled")

	// Explicit timeouts defeat Slowloris-style header/body starvation attacks
	// that pin connections to the server. ReadHeaderTimeout is the most
	// important — http.ListenAndServe leaves it as zero (unbounded). SSE
	// streams are long-lived so WriteTimeout/IdleTimeout are set generously
	// but bounded. ReadTimeout caps how long a slow client can take to send
	// the request body once the headers are in.
	httpServer := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           fleetmcp.NewHTTPHandler(mcpServer, config.MCPAuthToken, logger),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// SSE streams are long-lived, and a /mcp run_live_query POST is held
		// open for up to FLEET_LIVE_QUERY_REST_PERIOD; rely on idle/read timeouts.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fatal(ctx, logger, "server error", "err", err)
		}
	}()

	// Block until SIGINT/SIGTERM, then drain in-flight requests gracefully.
	<-ctx.Done()
	logger.InfoContext(ctx, "shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(ctx, "graceful shutdown failed", "err", err)
	}
}
