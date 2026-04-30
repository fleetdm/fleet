package main

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config holds the server configuration.
type Config struct {
	Port          string
	FleetBaseURL  string
	FleetAPIKey   string
	LogLevel      logrus.Level
	TLSSkipVerify bool   // FLEET_TLS_SKIP_VERIFY — skip TLS cert verification (unsafe; for dev only)
	TLSCAFile     string // FLEET_CA_FILE — path to PEM CA cert for self-signed Fleet instances
	MCPAuthToken  string // MCP_AUTH_TOKEN — bearer token required on all incoming MCP requests
}

// LoadConfig loads configuration from environment variables, falling back to .env if present.
//
// Secret resolution: FLEET_API_KEY and MCP_AUTH_TOKEN may be supplied either
// directly (via env var) or read from a file path in FLEET_API_KEY_FILE /
// MCP_AUTH_TOKEN_FILE. The *_FILE form is preferred so the secret never
// appears in process listings, shell history, or claude_desktop_config.json
// (which is readable by the user's UID and ends up in Time Machine backups).
// When both forms are set for the same secret, *_FILE wins and a warning is
// logged.
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		logrus.Debug("no .env file found, using environment variables")
	}

	logLevel, err := logrus.ParseLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	return &Config{
		Port:          getEnv("PORT", "8080"),
		FleetBaseURL:  getEnv("FLEET_BASE_URL", "https://localhost:8080"),
		FleetAPIKey:   resolveSecret("FLEET_API_KEY"),
		LogLevel:      logLevel,
		TLSSkipVerify: os.Getenv("FLEET_TLS_SKIP_VERIFY") == "true",
		TLSCAFile:     os.Getenv("FLEET_CA_FILE"),
		MCPAuthToken:  resolveSecret("MCP_AUTH_TOKEN"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// resolveSecret reads a secret value from either KEY_FILE (preferred — file
// path containing the secret) or KEY (direct env var). Trims surrounding
// whitespace including any trailing newline that text editors append. Returns
// "" if neither source provides a value.
//
// File reads are best-effort: a missing or unreadable KEY_FILE logs an error
// and falls back to the direct env var. This way a misconfigured *_FILE
// path doesn't take down a deployment that has the secret available via env
// (the conventional fallback used during migration to file-based secrets).
func resolveSecret(key string) string {
	fileKey := key + "_FILE"
	if path := strings.TrimSpace(os.Getenv(fileKey)); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			logrus.Errorf("failed to read %s=%s: %v — falling back to %s env var", fileKey, path, err, key)
		} else {
			val := strings.TrimSpace(string(data))
			if val == "" {
				logrus.Errorf("%s=%s is empty — falling back to %s env var", fileKey, path, key)
			} else {
				if os.Getenv(key) != "" {
					logrus.Warnf("%s and %s are both set — using %s (file form is preferred)", fileKey, key, fileKey)
				}
				return val
			}
		}
	}
	return strings.TrimSpace(os.Getenv(key))
}
