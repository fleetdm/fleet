//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/sigverify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	queries "github.com/fleetdm/fleet/v4/server/service/osquery_utils"
)

var preInstalled = []string{
	"firefox/darwin",
}

// verifyInstallerSignature runs before the install script. pkg installers
// carry their own signature and are verified (and Gatekeeper-assessed for
// notarization) directly; for dmg and zip installers the signature lives on
// the .app inside, which verifyInstalledApp checks after installation.
func verifyInstallerSignature(ctx context.Context, logger *slog.Logger, installerPath string, pin *maintained_apps.FMASignature) error {
	switch strings.ToLower(filepath.Ext(installerPath)) {
	case ".pkg", ".mpkg":
	default:
		logger.InfoContext(ctx, "Installer signature is verified on the installed app after installation for this format")
		return nil
	}

	res, err := sigverify.VerifyPkgSignature(ctx, installerPath)
	if err != nil {
		return fmt.Errorf("verifying pkg signature: %w", err)
	}
	return evaluateDarwinSignature(ctx, logger, res, pin)
}

// verifyInstalledApp verifies the installed .app bundle's code signature and
// Gatekeeper assessment for dmg/zip installers. It runs before
// postApplicationInstall adds a Gatekeeper exception and strips quarantine,
// so verification gates the bypass rather than being bypassed by it.
func verifyInstalledApp(ctx context.Context, logger *slog.Logger, appPath, installerPath string, pin *maintained_apps.FMASignature) error {
	switch strings.ToLower(filepath.Ext(installerPath)) {
	case ".pkg", ".mpkg":
		// Already verified pre-install by verifyInstallerSignature.
		return nil
	}
	if appPath == "" {
		// This check gates the Gatekeeper exception and quarantine stripping
		// in postApplicationInstall, so an undetected app path must fail
		// verification (report-only warns; enforce mode fails) rather than
		// silently pass.
		return errors.New("no installed .app bundle detected; cannot verify the app's signature for dmg/zip installers")
	}

	res, err := sigverify.VerifyAppBundle(ctx, appPath)
	if err != nil {
		return fmt.Errorf("verifying app bundle signature: %w", err)
	}
	return evaluateDarwinSignature(ctx, logger, res, pin)
}

// evaluateDarwinSignature compares an observed signature against the app's
// pin. It returns an error for the hard-fail conditions in the failure
// policy; the caller decides whether that fails validation (enforce mode) or
// warns (report-only).
func evaluateDarwinSignature(ctx context.Context, logger *slog.Logger, res *sigverify.DarwinResult, pin *maintained_apps.FMASignature) error {
	switch {
	case pin != nil && pin.Unsigned:
		if res.NoSignature {
			logger.InfoContext(ctx, "Installer is unsigned, as pinned")
			return nil
		}
		if !res.Verified {
			// A formerly-unsigned installer now carrying a broken or
			// untrusted signature is a tamper indicator, not a vendor
			// starting to sign.
			return fmt.Errorf("pin says unsigned but installer now carries an invalid signature: %s", res.Detail)
		}
		logger.WarnContext(ctx, fmt.Sprintf("Installer is now validly signed by %q but the pin says unsigned; update the pin", res.Identity))
		return nil
	case pin != nil:
		switch {
		case res.NoSignature:
			return fmt.Errorf("installer is unsigned but the pin expects team ID %s", pin.AppleTeamID)
		case !res.Verified:
			return fmt.Errorf("signature verification failed: %s", res.Detail)
		case res.TeamID != pin.AppleTeamID:
			return fmt.Errorf("signer identity changed: observed team ID %s, pinned %s", res.TeamID, pin.AppleTeamID)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Signature verified: signed by %q (matches pin)", res.Identity))
	default: // no pin
		switch {
		case res.NoSignature:
			return errors.New(`installer is unsigned and the app has no "unsigned" signature pin`)
		case !res.Verified:
			return fmt.Errorf("signature verification failed: %s", res.Detail)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Signature verified: signed by %q (no pin recorded yet)", res.Identity))
	}

	// Notarization: an accepted "Notarized Developer ID" Gatekeeper
	// assessment means Apple's automated malware analysis ran on these exact
	// bytes, and spctl consulted XProtect and Apple's revocation service.
	if pin != nil && pin.Notarized {
		if !res.NotarizationChecked {
			return errors.New("pin expects notarization but no Gatekeeper assessment could run")
		}
		if !res.Notarized {
			return fmt.Errorf("pin expects a notarization ticket but Gatekeeper assessed: %s", res.NotarizationDetail)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Notarization verified: %s", res.NotarizationDetail))
		return nil
	}
	if res.NotarizationChecked && !res.Notarized {
		logger.WarnContext(ctx, fmt.Sprintf("Installer was not assessed as notarized: %s", res.NotarizationDetail))
	}
	return nil
}

// scanInstallerForMalware is a no-op on macOS: the notarization enforcement
// in evaluateDarwinSignature is the malware layer (Apple's notary service
// scanned the bytes; spctl consults XProtect at assessment time).
func scanInstallerForMalware(_ context.Context, _ *slog.Logger, _ string) error {
	return nil
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

	appLogger.InfoContext(ctx, fmt.Sprintf("Attempting to remove quarantine for: '%s'", appPath))
	quarantineResult, err := removeAppQuarantine(appPath)

	appLogger.InfoContext(ctx, fmt.Sprintf("Quarantine output error: %v", quarantineResult.QuarantineOutputError))
	appLogger.InfoContext(ctx, fmt.Sprintf("Quarantine status: %s", quarantineResult.QuarantineStatus))
	if err != nil {
		return fmt.Errorf("Error removing app quarantine: %v. Attempting to continue", err)
	}
	return nil
}

type QuarantineResult struct {
	QuarantineOutputError error
	QuarantineStatus      string
}

// removeAppQuarantine adds a Gatekeeper exception and strips the quarantine
// attribute so the freshly installed app can launch in CI. The Gatekeeper
// assessment that used to be merely logged here now runs as a real gate in
// verifyInstalledApp BEFORE this bypass is applied, so this only eases
// installation of an already-verified binary.
func removeAppQuarantine(appPath string) (QuarantineResult, error) {
	var result QuarantineResult

	cmd := exec.Command("xattr", "-p", "com.apple.quarantine", appPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.QuarantineOutputError = fmt.Errorf("checking quarantine status: %v", err)
	}
	result.QuarantineStatus = fmt.Sprintf("Quarantine status: '%s'", strings.TrimSpace(string(output)))

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

func appExists(ctx context.Context, logger *slog.Logger, appName, uniqueAppIdentifier, appVersion, appPath string) (bool, error) {
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
			queries.MutateSoftwareOnIngestion(ctx, software, logger)
			result.Version = software.Version
			result.Name = software.Name

			logger.InfoContext(ctx, fmt.Sprintf("Found app: '%s' at %s, Version: %s, Bundled Version: %s", result.Name, result.Path, result.Version, result.BundledVersion))

			// OneDrive auto-updates immediately after installation, so the installed version
			// might be newer than the installer version. For OneDrive, we only verify that
			// the app exists rather than checking the version.
			if uniqueAppIdentifier == "com.microsoft.OneDrive" {
				logger.InfoContext(ctx, "OneDrive detected - skipping version check due to auto-update behavior")
				return true, nil
			}

			// GPG Suite's installer version (e.g., "2023.3") doesn't match the app bundle version
			// (e.g., "1.12" with bundled version "1800"). We only verify that the app exists
			// rather than checking the version.
			if uniqueAppIdentifier == "org.gpgtools.gpgkeychain" {
				logger.InfoContext(ctx, "GPG Suite detected - skipping version check due to version mismatch between installer and app bundle")
				return true, nil
			}

			// Adobe DNG Converter's version format includes build number in parentheses
			// (e.g., "18.0 (2389)") which doesn't match the installer version (e.g., "18.0")
			// Check if the version starts with the expected version to handle this case
			if uniqueAppIdentifier == "com.adobe.DNGConverter" {
				if strings.HasPrefix(result.Version, appVersion+" ") || strings.HasPrefix(result.Version, appVersion+"(") {
					logger.InfoContext(ctx, "Adobe DNG Converter detected - version matches with build number")
					return true, nil
				}
			}

			// Ableton Live's version format includes a build identifier in parentheses
			// (e.g., "12.4.1 (2026-05-20_fbe5fe99c9)") which doesn't match the installer
			// version (e.g., "12.4.1"). Check if the version starts with the expected
			// version to handle this case.
			if uniqueAppIdentifier == "com.ableton.live" {
				if strings.HasPrefix(result.Version, appVersion+" ") || strings.HasPrefix(result.Version, appVersion+"(") {
					logger.InfoContext(ctx, "Ableton Live detected - version matches with build identifier")
					return true, nil
				}
			}

			// WhatsApp: Homebrew sometimes reports a newer version than what's actually available.
			// If version doesn't match but app is installed, fall back to existence-only validation.
			if uniqueAppIdentifier == "net.whatsapp.WhatsApp" {
				if !checkVersionMatch(appVersion, result.Version, result.BundledVersion) {
					logger.InfoContext(ctx, "WhatsApp detected - version mismatch but app is installed, falling back to existence-only validation")
					return true, nil
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
					return true, nil
				}
			}

			// The Developer Edition cask version is the full beta ("153.0b13") but
			// the bundle reports only the base version ("153.0"); accept base+"b".
			if uniqueAppIdentifier == "org.mozilla.firefoxdeveloperedition" {
				if result.Version != "" && strings.HasPrefix(appVersion, result.Version+"b") {
					logger.InfoContext(ctx, "Firefox Developer Edition detected - cask version matches bundle base version with beta suffix")
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
