package main

import (
	"os"

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
}

// LoadConfig loads configuration from environment variables, falling back to .env if present.
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
		FleetAPIKey:   getEnv("FLEET_API_KEY", ""),
		LogLevel:      logLevel,
		TLSSkipVerify: os.Getenv("FLEET_TLS_SKIP_VERIFY") == "true",
		TLSCAFile:     os.Getenv("FLEET_CA_FILE"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
