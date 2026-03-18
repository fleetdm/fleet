package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// SeedFleet populates Fleet with a baseline set of standard saved queries.
// Run this once against a fresh Fleet instance to bootstrap common queries.
// Usage: set FLEET_BASE_URL and FLEET_API_KEY, then call SeedFleet() from a standalone main.
func SeedFleet() {
	_ = godotenv.Load()

	baseURL := os.Getenv("FLEET_BASE_URL")
	apiKey := os.Getenv("FLEET_API_KEY")
	if baseURL == "" || apiKey == "" {
		logrus.Fatal("FLEET_BASE_URL and FLEET_API_KEY must be set")
	}

	tlsSkipVerify := os.Getenv("FLEET_TLS_SKIP_VERIFY") == "true"
	caFile := os.Getenv("FLEET_CA_FILE")
	client := NewFleetClient(baseURL, apiKey, tlsSkipVerify, caFile)

	queries := []struct {
		Name        string
		Description string
		SQL         string
		Platform    string
	}{
		{
			Name:        "Standard: MacOS Admin Users",
			Description: "Lists all local users on macOS devices that have admin privileges.",
			SQL:         "SELECT * FROM users WHERE admin = 1;",
			Platform:    "darwin",
		},
		{
			Name:        "Standard: Windows Missing Updates",
			Description: "Lists Windows computers where updates have historically failed.",
			SQL:         "SELECT * FROM windows_update_history WHERE result_code = 'Failed';",
			Platform:    "windows",
		},
		{
			Name:        "Standard: Linux Running Containers",
			Description: "Lists all docker containers currently in a running state on Linux.",
			SQL:         "SELECT * FROM docker_containers WHERE state = 'running';",
			Platform:    "linux",
		},
		{
			Name:        "Standard: Universal OS Version",
			Description: "Retrieves the operating system name and version for all hosts.",
			SQL:         "SELECT * FROM os_version;",
			Platform:    "",
		},
	}

	success := 0
	for _, q := range queries {
		if _, err := client.CreateSavedQuery(q.Name, q.Description, q.SQL, q.Platform); err != nil {
			logrus.Errorf("failed to create %q: %v", q.Name, err)
			continue
		}
		logrus.Infof("created %q", q.Name)
		success++
	}

	logrus.Infof("seeding complete: %d/%d queries created", success, len(queries))
}
