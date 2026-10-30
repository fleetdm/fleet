package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Config holds the server configuration.
type Config struct {
	Port          string
	FleetBaseURL  string
	FleetAPIKey   string
	LogLevel      slog.Level
	TLSSkipVerify bool   // FLEET_TLS_SKIP_VERIFY — skip TLS cert verification (unsafe; for dev only)
	TLSCAFile     string // FLEET_CA_FILE — path to PEM CA cert for self-signed Fleet instances
	MCPAuthToken  string // MCP_AUTH_TOKEN — bearer token required on all incoming MCP requests
}

// LoadConfig loads configuration from environment variables.
//
// Secret resolution: FLEET_API_KEY and MCP_AUTH_TOKEN may be supplied via environment variables.
func LoadConfig() (*Config, error) {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(getEnv("LOG_LEVEL", "info"))); err != nil {
		return nil, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error: %w", err)
	}

	return &Config{
		Port:          getEnv("PORT", "8080"),
		FleetBaseURL:  getEnv("FLEET_BASE_URL", "https://localhost:8080"),
		FleetAPIKey:   os.Getenv("FLEET_API_KEY"),
		LogLevel:      logLevel,
		TLSSkipVerify: os.Getenv("FLEET_TLS_SKIP_VERIFY") == "true",
		TLSCAFile:     os.Getenv("FLEET_CA_FILE"),
		MCPAuthToken:  os.Getenv("MCP_AUTH_TOKEN"),
	}, nil
}

const defaultLiveQueryRestPeriod = 25 * time.Second

// maxSingleHostBlock caps the server-side wait the single-host timeout is
// derived from. FLEET_LIVE_QUERY_REST_PERIOD is also the multi-host campaign's
// wait budget, set far above anything a server blocks for (900s in render.yaml)
// so fleet-wide queries outlast asleep hosts; uncapped, that budget would let
// one stalled connection hang for 15 minutes. Fleet's own period must stay
// under the request timeout of any proxy in front of it, so real values are
// tens of seconds.
const maxSingleHostBlock = 120 * time.Second

// liveQueryWaits maps the standalone MCP's FLEET_LIVE_QUERY_REST_PERIOD to the
// multi-host campaign wait and the single-host block it expects from Fleet.
func liveQueryWaits(ctx context.Context, logger *slog.Logger, raw string) (campaignWait, singleHostBlock time.Duration) {
	period := defaultLiveQueryRestPeriod
	if raw != "" {
		// A non-positive period would make the client timeout <= 0, which
		// net/http treats as no timeout.
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			period = d
		} else {
			logger.WarnContext(ctx, "invalid FLEET_LIVE_QUERY_REST_PERIOD, defaulting to 25s", "value", raw)
		}
	}
	return period, min(period, maxSingleHostBlock)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
