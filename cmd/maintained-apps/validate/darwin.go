//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	queries "github.com/fleetdm/fleet/v4/server/service/osquery_utils"
)

var preInstalled = []string{
	"firefox/darwin",
}

func postApplicationInstall(ctx context.Context, appLogger *slog.Logger, appPath string) error {
	if appPath == "" {
		return nil
	}

	appLogger.InfoContext(ctx, fmt.Sprintf("Forcing LaunchServices refresh for: '%s'", appPath))
	err := forceLaunchServicesRefresh(appPath)
	if err != nil {
		return fmt.Errorf("Error forcing LaunchServices refresh: %v. Attempting to continue", err)
	}

	removed, err := removeAppQuarantine(appPath)
	if err != nil {
		return fmt.Errorf("Error removing app quarantine: %v. Attempting to continue", err)
	}
	if removed {
		appLogger.InfoContext(ctx, fmt.Sprintf("Removed quarantine attribute from: '%s'", appPath))
	}
	return nil
}

// Installers are downloaded with Go's HTTP client, which doesn't set
// com.apple.quarantine, so there is usually nothing to remove. spctl --add is
// not used: macOS 15+ no longer supports rule database changes (exit 4).
func removeAppQuarantine(appPath string) (bool, error) {
	// xattr -p exits 1 when the attribute isn't set.
	if err := exec.Command("xattr", "-p", "com.apple.quarantine", appPath).Run(); err != nil {
		return false, nil
	}

	cmd := exec.Command("sudo", "xattr", "-r", "-d", "com.apple.quarantine", appPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("removing quarantine attribute: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return true, nil
}

// assessGatekeeper returns an error when Gatekeeper would refuse to launch the
// installed app, e.g. it's unsigned, unnotarized, or its signature is broken.
func assessGatekeeper(ctx context.Context, appLogger *slog.Logger, appPath string) error {
	if appPath == "" {
		return nil
	}

	bundles := []string{appPath}
	if filepath.Ext(appPath) != ".app" {
		// Some installs add a folder (e.g. /Applications/KiCad) holding the app bundles.
		bundles = nil
		entries, _ := os.ReadDir(appPath)
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".app" {
				bundles = append(bundles, filepath.Join(appPath, entry.Name()))
			}
		}
	}
	if len(bundles) == 0 {
		appLogger.InfoContext(ctx, fmt.Sprintf("No app bundle found at '%s', skipping Gatekeeper assessment", appPath))
		return nil
	}

	var rejected []string
	for _, bundle := range bundles {
		output, err := exec.CommandContext(ctx, "spctl", "--assess", "--type", "execute", "-v", bundle).CombinedOutput()
		status := strings.ReplaceAll(strings.TrimSpace(string(output)), "\n", ", ")
		if err != nil {
			rejected = append(rejected, fmt.Sprintf("%s (%v)", status, err))
			continue
		}
		appLogger.InfoContext(ctx, fmt.Sprintf("Gatekeeper assessment: %s", status))
	}
	if len(rejected) > 0 {
		return fmt.Errorf("Gatekeeper rejected: %s", strings.Join(rejected, "; "))
	}
	return nil
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

// appExists reports whether the app is installed at the expected version. The
// second value is always empty because the app is always found in apps, the
// table the exists query reads.
func appExists(ctx context.Context, logger *slog.Logger, appName, uniqueAppIdentifier, appVersion, _, appPath, _ string) (bool, string, error) {
	execTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validateSqlInput(appName); err != nil {
		return false, "", fmt.Errorf("Invalid character found in appName: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(uniqueAppIdentifier); err != nil {
		return false, "", fmt.Errorf("Invalid character found in uniqueAppIdentifier: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(appPath); err != nil {
		return false, "", fmt.Errorf("Invalid character found in appPath: '%w'. Not executing query...", err)
	}

	logger.InfoContext(ctx, fmt.Sprintf("Looking for app: %s, version: %s", appName, appVersion))
	query := `
		SELECT
		  COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app') ) AS name,
		  path,
		  bundle_short_version,
		  bundle_version
		FROM apps
		WHERE 
		bundle_identifier LIKE '%` + uniqueAppIdentifier + `%' OR
		LOWER(COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app'))) LIKE LOWER('%` + appName + `%')
	`
	if appPath != "" {
		query += fmt.Sprintf(" OR path LIKE '%%%s%%'", appPath)
	}
	cmd := exec.CommandContext(execTimeout, "osqueryi", "--json", query)
	output, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		Version        string `json:"bundle_short_version"`
		BundledVersion string `json:"bundle_version"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		return false, "", fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			software := &fleet.Software{
				Name:             result.Name,
				Version:          result.Version,
				BundleIdentifier: uniqueAppIdentifier,
				Source:           "apps",
			}
			queries.MutateSoftwareOnIngestion(ctx, software, logger)
			result.Version = software.Version
			result.Name = software.Name

			logger.InfoContext(ctx, fmt.Sprintf("Found app: '%s' at %s, Version: %s, Bundled Version: %s", result.Name, result.Path, result.Version, result.BundledVersion))

			// OneDrive auto-updates immediately after installation, so the installed version
			// might be newer than the installer version. For OneDrive, we only verify that
			// the app exists rather than checking the version.
			if uniqueAppIdentifier == "com.microsoft.OneDrive" {
				if !checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
					logger.InfoContext(ctx, "OneDrive detected - version mismatch but app is installed, falling back to existence-only validation")
					return true, "", nil
				}
			}

			// GPG Suite's installer version (e.g., "2023.3") doesn't match the app bundle version
			// (e.g., "1.12" with bundled version "1800"). We only verify that the app exists
			// rather than checking the version.
			if uniqueAppIdentifier == "org.gpgtools.gpgkeychain" {
				logger.InfoContext(ctx, "GPG Suite detected - skipping version check due to version mismatch between installer and app bundle")
				return true, "", nil
			}

			// WezTerm's bundle reports a placeholder version ("0.1.0" / "1") in every
			// release, which never matches the cask's date-based version.
			if uniqueAppIdentifier == "com.github.wez.wezterm" && result.Path == "/Applications/WezTerm.app" {
				logger.InfoContext(ctx, "WezTerm detected - skipping version check because the app bundle carries a placeholder version")
				return true, "", nil
			}

			// The name "Dia" also matches Apple Diagnostics, whose bundle version "1"
			// prefix-matches Dia's version, so only count Dia's own bundle.
			if uniqueAppIdentifier == "company.thebrowser.dia" && result.Path != "/Applications/Dia.app" {
				continue
			}

			// Adobe DNG Converter's version format includes build number in parentheses
			// (e.g., "18.0 (2389)") which doesn't match the installer version (e.g., "18.0")
			// Check if the version starts with the expected version to handle this case
			if uniqueAppIdentifier == "com.adobe.DNGConverter" {
				if strings.HasPrefix(result.Version, appVersion+" ") || strings.HasPrefix(result.Version, appVersion+"(") {
					logger.InfoContext(ctx, "Adobe DNG Converter detected - version matches with build number")
					return true, "", nil
				}
			}

			// Ableton Live's version format includes a build identifier in parentheses
			// (e.g., "12.4.1 (2026-05-20_fbe5fe99c9)") which doesn't match the installer
			// version (e.g., "12.4.1"). Check if the version starts with the expected
			// version to handle this case.
			if uniqueAppIdentifier == "com.ableton.live" {
				if strings.HasPrefix(result.Version, appVersion+" ") || strings.HasPrefix(result.Version, appVersion+"(") {
					logger.InfoContext(ctx, "Ableton Live detected - version matches with build identifier")
					return true, "", nil
				}
			}

			// WhatsApp: Homebrew sometimes reports a newer version than what's actually available.
			// If version doesn't match but app is installed, fall back to existence-only validation.
			if uniqueAppIdentifier == "net.whatsapp.WhatsApp" {
				if !checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
					logger.InfoContext(ctx, "WhatsApp detected - version mismatch but app is installed, falling back to existence-only validation")
					return true, "", nil
				}
			}

			// Logi Tune: the installer URL always serves the latest release, while the Homebrew
			// cask version lags behind (its livecheck scrapes a Logitech support article that is
			// updated less often than the download). The installed version is therefore newer
			// than the manifest version. If version doesn't match but app is installed, fall
			// back to existence-only validation.
			if uniqueAppIdentifier == "com.logitech.logitune" {
				if !checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
					logger.InfoContext(ctx, "Logi Tune detected - version mismatch but app is installed, falling back to existence-only validation")
					return true, "", nil
				}
			}

			// The Developer Edition cask version is the full beta ("153.0b13") but
			// the bundle reports only the base version ("153.0"); accept base+"b".
			if uniqueAppIdentifier == "org.mozilla.firefoxdeveloperedition" {
				if result.Version != "" && strings.HasPrefix(appVersion, result.Version+"b") {
					logger.InfoContext(ctx, "Firefox Developer Edition detected - cask version matches bundle base version with beta suffix")
					return true, "", nil
				}
			}

			// Check various version matching strategies
			if checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
				return true, "", nil
			}
		}
	}

	return false, "", nil
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
