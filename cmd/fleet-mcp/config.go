package main

import (
	"fmt"
	"log/slog"
	"os"
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

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
