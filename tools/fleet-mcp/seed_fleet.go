package main

import (
	"context"

	"github.com/sirupsen/logrus"
)

// SeedFleet populates Fleet with a baseline set of standard saved reports (formerly "queries").
// Run this once against a fresh Fleet instance to bootstrap common reports.
// Invoke via the -seed flag on the fleet-mcp binary; the caller supplies the
// already-loaded Config and FleetClient so this function does not duplicate
// environment loading or client construction.
func SeedFleet(config *Config, client *FleetClient) {
	_ = config // reserved for future use (e.g. choosing query sets per env)

	queries := []struct {
		Name        string
		Description string
		SQL         string
		Platform    string
	}{
		{
			Name:        "Standard: MacOS Admin Users",
			Description: "Lists all local users on macOS devices that have admin privileges.",
			SQL: "SELECT u.username, u.uid, u.gid, u.directory, u.shell " +
				"FROM users u " +
				"JOIN user_groups ug ON u.uid = ug.uid " +
				"JOIN groups g ON ug.gid = g.gid " +
				"WHERE g.groupname = 'admin';",
			Platform: "darwin",
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
		// Seed queries are intentionally Global — they are demo/sample data
		// not tied to any specific team.
		if _, err := client.CreateSavedQuery(context.Background(), q.Name, q.Description, q.SQL, q.Platform, nil); err != nil {
			logrus.Errorf("failed to create %q: %v", q.Name, err)
			continue
		}
		logrus.Infof("created %q", q.Name)
		success++
	}

	logrus.Infof("seeding complete: %d/%d queries created", success, len(queries))
}
