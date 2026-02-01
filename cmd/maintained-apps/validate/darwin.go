//go:build darwin

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
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	queries "github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var preInstalled = []string{
	"firefox/darwin",
}

func postApplicationInstall(appLogger kitlog.Logger, appPath string) error {
	if appPath == "" {
		return nil
	}

	level.Info(appLogger).Log("msg", fmt.Sprintf("Forcing LaunchServices refresh for: '%s'", appPath))
	err := forceLaunchServicesRefresh(appPath)
	if err != nil {
		return fmt.Errorf("Error forcing LaunchServices refresh: %v. Attempting to continue", err)
	}

	level.Info(appLogger).Log("msg", fmt.Sprintf("Attempting to remove quarantine for: '%s'", appPath))
	quarantineResult, err := removeAppQuarantine(appPath)

	level.Info(appLogger).Log("msg", fmt.Sprintf("Quarantine output error: %v", quarantineResult.QuarantineOutputError))
	level.Info(appLogger).Log("msg", fmt.Sprintf("Quarantine status: %s", quarantineResult.QuarantineStatus))
	level.Info(appLogger).Log("msg", fmt.Sprintf("Spctl output error: %v", quarantineResult.SpctlOutputError))
	level.Info(appLogger).Log("msg", fmt.Sprintf("spctl status: %s", quarantineResult.SpctlStatus))
	if err != nil {
		return fmt.Errorf("Error removing app quarantine: %v. Attempting to continue", err)
	}
	return nil
}

type QuarantineResult struct {
	QuarantineOutputError error
	QuarantineStatus      string
	SpctlOutputError      error
	SpctlStatus           string
}

func removeAppQuarantine(appPath string) (QuarantineResult, error) {
	var result QuarantineResult

	cmd := exec.Command("xattr", "-p", "com.apple.quarantine", appPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.QuarantineOutputError = fmt.Errorf("checking quarantine status: %v", err)
	}
	result.QuarantineStatus = fmt.Sprintf("Quarantine status: '%s'", strings.TrimSpace(string(output)))
	cmd = exec.Command("spctl", "-a", "-v", appPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		result.SpctlOutputError = fmt.Errorf("checking spctl status: %v", err)
	}
	result.SpctlStatus = fmt.Sprintf("spctl status: '%s'", strings.TrimSpace(string(output)))

	cmd = exec.Command("sudo", "spctl", "--add", appPath)
	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("adding app to quarantine exceptions: %w", err)
	}

	cmd = exec.Command("sudo", "xattr", "-r", "-d", "com.apple.quarantine", appPath)
	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("removing quarantine attribute: %w", err)
	}

	return result, nil
}

func forceLaunchServicesRefresh(appPath string) error {
	cmd := exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-f", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forcing LaunchServices refresh: %w", err)
	}
	time.Sleep(2 * time.Second)
	return nil
}

// normalizeVersion removes common suffixes from version strings
func normalizeVersion(v string) string {
	suffixes := []string{"-latest", "-beta", "-alpha", "-rc", "-pre"}
	normalized := v
	for _, suffix := range suffixes {
		normalized = strings.TrimSuffix(normalized, suffix)
	}
	return normalized
}

// checkVersionMatch checks if the expected version matches any of the found versions
// using various matching strategies: exact match, normalization, concatenation, and prefix matching
func checkVersionMatch(expectedVersion, foundVersion, foundBundledVersion string) bool {
	// Check exact matches first (no normalization needed)
	if expectedVersion == foundVersion || expectedVersion == foundBundledVersion {
		return true
	}

	// Only normalize if exact match failed (lazy normalization)
	normalizedExpected := normalizeVersion(expectedVersion)
	normalizedFound := normalizeVersion(foundVersion)
	normalizedBundled := normalizeVersion(foundBundledVersion)

	// Check normalized exact matches
	if normalizedExpected == normalizedFound || normalizedExpected == normalizedBundled {
		return true
	}

	// Check if expected version is a concatenation of short version + bundled version
	// This handles cases like "1.4.230579" = "1.4.2" + "30579" or "1.4.2.30579" = "1.4.2" + "." + "30579"
	// Only check concatenation if the expected version is longer than the short version alone,
	// which indicates it might be a concatenation (avoids false positives)
	if foundVersion != "" && foundBundledVersion != "" && len(expectedVersion) > len(foundVersion) {
		// Try direct concatenation (no separator)
		concatenated := foundVersion + foundBundledVersion
		if expectedVersion == concatenated {
			return true
		}
		// Check normalized concatenation
		normalizedConcatenated := normalizedFound + normalizedBundled
		if normalizedExpected == normalizedConcatenated {
			return true
		}
		// Try concatenation with dot separator
		concatenatedWithDot := foundVersion + "." + foundBundledVersion
		if expectedVersion == concatenatedWithDot {
			return true
		}
		normalizedConcatenatedWithDot := normalizedFound + "." + normalizedBundled
		if normalizedExpected == normalizedConcatenatedWithDot {
			return true
		}
	}

	// Check if found version starts with expected version (handles suffixes like ".CE")
	// This handles cases where the app version is "8.0.44.CE" but expected is "8.0.44"
	if strings.HasPrefix(foundVersion, expectedVersion+".") ||
		strings.HasPrefix(foundBundledVersion, expectedVersion+".") {
		return true
	}
	if strings.HasPrefix(normalizedFound, normalizedExpected+".") ||
		strings.HasPrefix(normalizedBundled, normalizedExpected+".") {
		return true
	}

	// Check if expected version starts with found version (handles cases where osquery reports shorter version)
	// This handles cases where expected is "2025.2.1.8" but osquery reports "2025.2"
	if strings.HasPrefix(expectedVersion, foundVersion+".") {
		return true
	}
	if strings.HasPrefix(normalizedExpected, normalizedFound+".") {
		return true
	}
	// Also check bundled version for prefix matches
	if strings.HasPrefix(expectedVersion, foundBundledVersion+".") {
		return true
	}
	if strings.HasPrefix(normalizedExpected, normalizedBundled+".") {
		return true
	}

	return false
}

func appExists(ctx context.Context, logger kitlog.Logger, appName, uniqueAppIdentifier, appVersion, appPath string) (bool, error) {
	execTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validateSqlInput(appName); err != nil {
		return false, fmt.Errorf("Invalid character found in appName: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(uniqueAppIdentifier); err != nil {
		return false, fmt.Errorf("Invalid character found in uniqueAppIdentifier: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(appPath); err != nil {
		return false, fmt.Errorf("Invalid character found in appPath: '%w'. Not executing query...", err)
	}

	level.Info(logger).Log("msg", fmt.Sprintf("Looking for app: %s, version: %s\n", appName, appVersion))
	// Escape single quotes for SQLite by doubling them to prevent SQL injection
	escapedAppIdentifier := strings.ReplaceAll(uniqueAppIdentifier, "'", "''")
	escapedAppName := strings.ReplaceAll(appName, "'", "''")
	query := `
		SELECT
		  COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app') ) AS name,
		  path,
		  bundle_short_version,
		  bundle_version
		FROM apps
		WHERE
		bundle_identifier LIKE '%` + escapedAppIdentifier + `%' OR
		LOWER(COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app'))) LIKE LOWER('%` + escapedAppName + `%')
	`
	if appPath != "" {
		escapedAppPath := strings.ReplaceAll(appPath, "'", "''")
		query += fmt.Sprintf(" OR path LIKE '%%%s%%'", escapedAppPath)
	}
	cmd := exec.CommandContext(execTimeout, "osqueryi", "--json", query)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		Version        string `json:"bundle_short_version"`
		BundledVersion string `json:"bundle_version"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		return false, fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			software := &fleet.Software{
				Name:             result.Name,
				Version:          result.Version,
				BundleIdentifier: uniqueAppIdentifier,
				Source:           "apps",
			}
			queries.MutateSoftwareOnIngestion(software, logger)
			result.Version = software.Version
			result.Name = software.Name

			level.Info(logger).Log("msg", fmt.Sprintf("Found app: '%s' at %s, Version: %s, Bundled Version: %s", result.Name, result.Path, result.Version, result.BundledVersion))

			// OneDrive auto-updates immediately after installation, so the installed version
			// might be newer than the installer version. For OneDrive, we only verify that
			// the app exists rather than checking the version.
			if uniqueAppIdentifier == "com.microsoft.OneDrive" {
				level.Info(logger).Log("msg", "OneDrive detected - skipping version check due to auto-update behavior")
				return true, nil
			}

			// GPG Suite's installer version (e.g., "2023.3") doesn't match the app bundle version
			// (e.g., "1.12" with bundled version "1800"). We only verify that the app exists
			// rather than checking the version.
			if uniqueAppIdentifier == "org.gpgtools.gpgkeychain" {
				level.Info(logger).Log("msg", "GPG Suite detected - skipping version check due to version mismatch between installer and app bundle")
				return true, nil
			}

			// Adobe DNG Converter's version format includes build number in parentheses
			// (e.g., "18.0 (2389)") which doesn't match the installer version (e.g., "18.0")
			// Check if the version starts with the expected version to handle this case
			if uniqueAppIdentifier == "com.adobe.DNGConverter" {
				if strings.HasPrefix(result.Version, appVersion+" ") || strings.HasPrefix(result.Version, appVersion+"(") {
					level.Info(logger).Log("msg", "Adobe DNG Converter detected - version matches with build number")
					return true, nil
				}
			}

			// WhatsApp: Homebrew sometimes reports a newer version than what's actually available.
			// If version doesn't match but app is installed, fall back to existence-only validation.
			if uniqueAppIdentifier == "net.whatsapp.WhatsApp" {
				if !checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
					level.Info(logger).Log("msg", "WhatsApp detected - version mismatch but app is installed, falling back to existence-only validation")
					return true, nil
				}
			}

			// Check various version matching strategies
			if checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
				return true, nil
			}
		}
	}

	return false, nil
}

func executeScript(cfg *Config, scriptContents string) (string, error) {
	scriptExtension := ".sh"
	scriptPath := filepath.Join(cfg.tmpDir, "script"+scriptExtension)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, exitCode, err := scripts.ExecCmd(ctx, scriptPath, cfg.env)
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
