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

			// Microsoft Excel installed via Office Deployment Tool (ODT) has a different version
			// than the ODT installer version. The ODT version (e.g., 16.0.19426.20170) doesn't match
			// the installed Excel version (e.g., the Office/Microsoft 365 version).
			// We only verify that Excel exists rather than checking the version.
			if appName == "Microsoft Excel" {
				level.Info(logger).Log("msg", "Microsoft Excel detected - skipping version check (ODT version doesn't match installed Excel version)")
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
			// Check if expected version starts with found version (handles cases where osquery reports shorter version)
			// This handles cases where expected is "6.4.0" but osquery reports "6.4"
			if strings.HasPrefix(appVersion, result.Version+".") {
				return true, nil
			}
		}
	}

	// For AppX packages, check if the package is provisioned
	// Provisioned packages don't show up in the programs table until a user logs in
	// Since unique_identifier should match DisplayName, use it for exact match
	if uniqueIdentifier == "" {
		return false, nil
	}

	// Search by DisplayName using exact match (unique_identifier should match DisplayName)
	provisionedQuery := fmt.Sprintf(`Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -eq '%s' } | Select-Object -First 1 | ConvertTo-Json -Depth 5`, uniqueIdentifier)
	cmd = exec.CommandContext(execTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", provisionedQuery)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}

	if len(output) > 0 {
		outputStr := strings.TrimSpace(string(output))
		// Handle case where PowerShell returns an empty array []
		if outputStr == "[]" || outputStr == "null" {
			return false, nil
		}

		var provisioned struct {
			DisplayName string `json:"DisplayName"`
			PackageName string `json:"PackageName"`
			Version     string `json:"Version"` // Version is a string like "11.2.1495.0"
		}
		if err := json.Unmarshal([]byte(outputStr), &provisioned); err != nil {
			return false, nil
		}

		if provisioned.DisplayName != "" || provisioned.PackageName != "" {
			provisionedVersion := provisioned.Version
			level.Info(logger).Log("msg", fmt.Sprintf("Found provisioned AppX package: '%s', Version: %s", provisioned.DisplayName, provisionedVersion))

			// Normalize both versions for comparison
			normalizedProvisioned := normalizeVersion(provisionedVersion)
			normalizedExpected := normalizeVersion(appVersion)

			// Check if version matches (exact or prefix match)
			if normalizedProvisioned == normalizedExpected ||
				strings.HasPrefix(normalizedProvisioned, normalizedExpected+".") ||
				strings.HasPrefix(normalizedExpected, normalizedProvisioned+".") ||
				provisionedVersion == appVersion ||
				strings.HasPrefix(provisionedVersion, appVersion+".") ||
				strings.HasPrefix(appVersion, provisionedVersion+".") {
				return true, nil
			}
		}
	}

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
