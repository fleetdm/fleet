//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	queries "github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var preInstalled = []string{}

func postApplicationInstall(_ kitlog.Logger, _ string) error {
	return nil
}

// normalizeVersion normalizes version strings for comparison
// Handles cases like "11.2.1495.0" vs "11.2.1495" by padding with zeros
func normalizeVersion(version string) string {
	parts := strings.Split(version, ".")
	// Ensure we have at least 4 parts (Major.Minor.Build.Revision)
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	// Trim to 4 parts max
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, ".")
}

func appExists(ctx context.Context, logger kitlog.Logger, appName, uniqueIdentifier, appVersion, appPath string) (bool, error) {
	execTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validateSqlInput(appName); err != nil {
		return false, fmt.Errorf("Invalid character found in appName: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(appPath); err != nil {
		return false, fmt.Errorf("Invalid character found in appPath: '%w'. Not executing query...", err)
	}

	level.Info(logger).Log("msg", fmt.Sprintf("Looking for app: %s, version: %s", appName, appVersion))
	query := `
		SELECT name, install_location, version 
		FROM programs
		WHERE
		LOWER(name) LIKE LOWER('%` + appName + `%')
	`
	if appPath != "" {
		query += fmt.Sprintf(" OR install_location LIKE '%%%s%%'", appPath)
	}
	cmd := exec.CommandContext(execTimeout, "osqueryi", "--json", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("osquery output: %s", string(output)))
		return false, fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name            string `json:"name"`
		InstallLocation string `json:"install_location"`
		Version         string `json:"version"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("osquery output: %s", string(output)))
		return false, fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			software := &fleet.Software{
				Name:    result.Name,
				Version: result.Version,
				Source:  "programs",
			}
			queries.MutateSoftwareOnIngestion(software, logger)
			result.Version = software.Version
			result.Name = software.Name

			level.Info(logger).Log("msg", fmt.Sprintf("Found app: '%s' at %s, Version: %s", result.Name, result.InstallLocation, result.Version))

			// Sublime Text's Inno Setup installer may not write version to registry properly
			// If app is found but version is empty, check if it's Sublime Text and skip version check
			if appName == "Sublime Text" && result.Version == "" {
				level.Info(logger).Log("msg", "Sublime Text detected with empty version - skipping version check (installer may not write version to registry)")
				return true, nil
			}

			// Check exact match first
			if result.Version == appVersion {
				return true, nil
			}
			// Check if found version starts with expected version (handles suffixes like ".0")
			// This handles cases where the app version is "3.5.4.0" but expected is "3.5.4"
			if strings.HasPrefix(result.Version, appVersion+".") {
				return true, nil
			}
		}
	}

	// For AppX packages (like Company Portal), check if the package is provisioned
	// Provisioned packages don't show up in the programs table until a user logs in
	// Try multiple search strategies: by unique identifier, app name, and common AppX package names
	level.Info(logger).Log("msg", "App not found in programs table, checking for provisioned AppX package...")
	
	// Build search terms - try unique identifier, app name, and package identifier patterns
	searchTerms := []string{}
	if uniqueIdentifier != "" {
		searchTerms = append(searchTerms, uniqueIdentifier)
	}
	searchTerms = append(searchTerms, appName)
	// Add common variations for Company Portal
	if strings.Contains(strings.ToLower(appName), "company portal") || strings.Contains(strings.ToLower(uniqueIdentifier), "company portal") {
		searchTerms = append(searchTerms, "Company Portal", "Microsoft.CompanyPortal", "Microsoft Company Portal")
	}
	
	// First, list all provisioned packages for debugging
	listQuery := `Get-AppxProvisionedPackage -Online | Select-Object DisplayName, PackageName, Version | ConvertTo-Json -Depth 5`
	listCmd := exec.CommandContext(execTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", listQuery)
	listOutput, _ := listCmd.CombinedOutput()
	if len(listOutput) > 0 {
		level.Info(logger).Log("msg", fmt.Sprintf("All provisioned packages: %s", string(listOutput)))
	}
	
	// Try to find the package using various search terms
	for _, searchTerm := range searchTerms {
		if searchTerm == "" {
			continue
		}
		// Search by DisplayName or PackageName (case-insensitive)
		provisionedQuery := fmt.Sprintf(`Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -like '*%s*' -or $_.PackageName -like '*%s*' } | Select-Object -First 1 | ConvertTo-Json -Depth 5`, searchTerm, searchTerm)
		cmd := exec.CommandContext(execTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", provisionedQuery)
		output, err := cmd.CombinedOutput()
		if err != nil {
			level.Info(logger).Log("msg", fmt.Sprintf("Error checking for provisioned package with search term '%s': %v", searchTerm, err))
			continue
		}
		if len(output) > 0 {
			var provisioned struct {
				DisplayName string `json:"DisplayName"`
				PackageName string `json:"PackageName"`
				Version     struct {
					Major    int `json:"Major"`
					Minor    int `json:"Minor"`
					Build    int `json:"Build"`
					Revision int `json:"Revision"`
				} `json:"Version"`
			}
			if err := json.Unmarshal(output, &provisioned); err == nil && (provisioned.DisplayName != "" || provisioned.PackageName != "") {
				// Format version as "Major.Minor.Build.Revision"
				provisionedVersion := fmt.Sprintf("%d.%d.%d.%d",
					provisioned.Version.Major,
					provisioned.Version.Minor,
					provisioned.Version.Build,
					provisioned.Version.Revision)
				level.Info(logger).Log("msg", fmt.Sprintf("Found provisioned AppX package: '%s' (Package: %s), Version: %s", provisioned.DisplayName, provisioned.PackageName, provisionedVersion))
				
				// Normalize both versions for comparison
				normalizedProvisioned := normalizeVersion(provisionedVersion)
				normalizedExpected := normalizeVersion(appVersion)
				
				// Check if version matches (exact or prefix match)
				// Also check if expected version starts with provisioned version (handles cases where expected is "11.2.1495" but provisioned is "11.2.1495.0")
				if normalizedProvisioned == normalizedExpected ||
					strings.HasPrefix(normalizedProvisioned, normalizedExpected+".") ||
					strings.HasPrefix(normalizedExpected, normalizedProvisioned+".") ||
					provisionedVersion == appVersion ||
					strings.HasPrefix(provisionedVersion, appVersion+".") ||
					strings.HasPrefix(appVersion, provisionedVersion+".") {
					level.Info(logger).Log("msg", fmt.Sprintf("Provisioned AppX package version matches expected version (provisioned: %s, expected: %s)", provisionedVersion, appVersion))
					return true, nil
				}
				level.Info(logger).Log("msg", fmt.Sprintf("Provisioned version '%s' (normalized: %s) does not match expected version '%s' (normalized: %s)", provisionedVersion, normalizedProvisioned, appVersion, normalizedExpected))
				// Found the package but version doesn't match - return false
				return false, nil
			}
		}
	}
	
	level.Info(logger).Log("msg", "No provisioned AppX package found matching search terms")

	return false, nil
}

func executeScript(cfg *Config, scriptContents string) (string, error) {
	scriptExtension := ".ps1"
	scriptPath := filepath.Join(cfg.tmpDir, "script"+scriptExtension)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use custom execution with non-interactive flags for Windows
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.WaitDelay = 1 * time.Minute
	cmd.Env = cfg.env
	cmd.Dir = filepath.Dir(scriptPath)

	output, err := cmd.CombinedOutput()

	exitCode := -1

	// Only set exitCode if process completed and context wasn't cancelled
	if cmd.ProcessState != nil {
		// see orbit/pkg/scripts/exec_windows.go
		// https://en.wikipedia.org/wiki/Exit_status#Windows
		exitCode = int(int32(cmd.ProcessState.ExitCode())) // nolint:gosec
	}

	result := fmt.Sprintf(`
--------------------
%s
--------------------`, string(output))

	if err != nil {
		return result, err
	}
	if exitCode != 0 {
		return result, fmt.Errorf("script execution failed with exit code %d: %s", exitCode, string(output))
	}
	return result, nil
}
