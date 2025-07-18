//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

func postApplicationInstall(appPath string) error {
	return nil
}

func doesAppExists(appName, uniqueAppIdentifier, appVersion, appPath string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Looking for app: %s, version: %s\n", appName, appVersion)
	query := `
		SELECT name, install_location, version 
		FROM programs
		WHERE
		name LIKE '%` + appName + `%'
	`
	if appPath != "" {
		query += fmt.Sprintf(" OR install_location LIKE '%%%s%%'", appPath)
	}
	cmd := exec.CommandContext(ctx, "osqueryi", "--json", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("osquery output: %s\n", string(output))
		return false, fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name            string `json:"name"`
		InstallLocation string `json:"install_location"`
		Version         string `json:"version"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		fmt.Printf("osquery output: %s\n", string(output))
		return false, fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			fmt.Printf("Found app: '%s' at %s, Version: %s\n", result.Name, result.InstallLocation, result.Version)
			if result.Version == appVersion {
				return true, nil
			}
		}
	}

	return false, nil
}
