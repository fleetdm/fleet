//go:build windows

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
	"github.com/fleetdm/fleet/v4/server/fleet"
	queries "github.com/fleetdm/fleet/v4/server/service/osquery_utils"
)

var preInstalled = []string{}

func postApplicationInstall(_ context.Context, _ *slog.Logger, _ string) error {
	return nil
}

// authenticodeSignature is the result of Get-AuthenticodeSignature, which
// verifies the full chain (including revocation) against the Windows trust
// store — the authoritative Authenticode check.
type authenticodeSignature struct {
	Status        string `json:"Status"`
	StatusMessage string `json:"StatusMessage"`
	Subject       string `json:"Subject"`
}

// verifyInstallerSignature runs before the install script: it verifies the
// installer's Authenticode signature and compares the signer's subject CN
// against the pin in the app's input JSON.
func verifyInstallerSignature(ctx context.Context, logger *slog.Logger, installerPath string, pin *maintained_apps.FMASignature) error {
	sig, err := getAuthenticodeSignature(ctx, installerPath)
	if err != nil {
		return fmt.Errorf("running Get-AuthenticodeSignature: %w", err)
	}

	observedCN := sigverify.SubjectCNFromX500DN(sig.Subject)

	switch {
	case pin != nil && pin.Unsigned:
		if sig.Status == "NotSigned" {
			logger.InfoContext(ctx, "Installer is unsigned, as pinned")
			return nil
		}
		logger.WarnContext(ctx, fmt.Sprintf("Installer is now signed by %q but the pin says unsigned; update the pin", observedCN))
		return nil
	case pin != nil:
		switch {
		case sig.Status == "NotSigned":
			return fmt.Errorf("installer is unsigned but the pin expects signer %v", pin.SubjectCNs)
		case sig.Status != "Valid":
			return fmt.Errorf("Authenticode signature is not valid: %s (%s)", sig.Status, sig.StatusMessage)
		case !pin.MatchesSubjectCN(observedCN):
			return fmt.Errorf("signer identity changed: observed %q, pinned %v", observedCN, pin.SubjectCNs)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Authenticode signature verified: signed by %q (matches pin)", observedCN))
	default: // no pin
		switch {
		case sig.Status == "NotSigned":
			return errors.New(`installer is unsigned and the app has no "unsigned" signature pin`)
		case sig.Status != "Valid":
			return fmt.Errorf("Authenticode signature is not valid: %s (%s)", sig.Status, sig.StatusMessage)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Authenticode signature verified: signed by %q (no pin recorded yet)", observedCN))
	}
	return nil
}

func getAuthenticodeSignature(ctx context.Context, installerPath string) (*authenticodeSignature, error) {
	execTimeout, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Single-quote the path for PowerShell (doubling embedded quotes); the
	// path is under our own temp directory.
	quoted := "'" + strings.ReplaceAll(installerPath, "'", "''") + "'"
	psCommand := fmt.Sprintf(
		`$sig = Get-AuthenticodeSignature -LiteralPath %s; [PSCustomObject]@{Status = $sig.Status.ToString(); StatusMessage = [string]$sig.StatusMessage; Subject = if ($sig.SignerCertificate) { $sig.SignerCertificate.Subject } else { '' }} | ConvertTo-Json -Compress`,
		quoted,
	)
	out, err := exec.CommandContext(execTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCommand).Output()
	if err != nil {
		return nil, fmt.Errorf("executing PowerShell: %w", err)
	}

	var sig authenticodeSignature
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &sig); err != nil {
		return nil, fmt.Errorf("parsing Get-AuthenticodeSignature output: %w", err)
	}
	return &sig, nil
}

// verifyInstalledApp is a no-op on Windows: the Authenticode check runs on
// the installer file itself before installation.
func verifyInstalledApp(_ context.Context, _ *slog.Logger, _, _ string, _ *maintained_apps.FMASignature) error {
	return nil
}

// scanInstallerForMalware runs a Microsoft Defender on-demand scan of the
// installer. GitHub-hosted runners keep Defender in passive mode, but the
// engine and MpCmdRun.exe are present. A detection returns an error (hard
// fail once FMA_SCAN_ENFORCE is set); an unavailable or incomplete scan only
// warns, so the check is best-effort by design.
func scanInstallerForMalware(ctx context.Context, logger *slog.Logger, installerPath string) error {
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	mpCmdRun := filepath.Join(programFiles, "Windows Defender", "MpCmdRun.exe")
	if _, err := os.Stat(mpCmdRun); err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("Microsoft Defender not found at %s; skipping malware scan", mpCmdRun))
		return nil
	}

	execTimeout, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	logger.InfoContext(ctx, "Scanning installer with Microsoft Defender...")
	cmd := exec.CommandContext(execTimeout, mpCmdRun,
		"-Scan", "-ScanType", "3", "-File", installerPath, "-DisableRemediation")
	out, err := cmd.CombinedOutput()

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = int(int32(cmd.ProcessState.ExitCode())) // nolint:gosec
	}

	switch exitCode {
	case 0:
		logger.InfoContext(ctx, "Microsoft Defender scan found no threats")
		return nil
	case 2:
		return fmt.Errorf("Microsoft Defender detected a threat in the installer: %s", strings.TrimSpace(string(out)))
	default:
		// Scan couldn't complete (stale definitions, passive-mode quirks);
		// warn rather than fail so a runner-image issue doesn't block updates.
		logger.WarnContext(ctx, fmt.Sprintf("Microsoft Defender scan did not complete (exit code %d, err %v): %s", exitCode, err, strings.TrimSpace(string(out))))
		return nil
	}
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

func appExists(ctx context.Context, logger *slog.Logger, appName, uniqueIdentifier, appVersion, appPath string) (bool, error) {
	execTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := validateSqlInput(appName); err != nil {
		return false, fmt.Errorf("Invalid character found in appName: '%w'. Not executing query...", err)
	}
	if err := validateSqlInput(appPath); err != nil {
		return false, fmt.Errorf("Invalid character found in appPath: '%w'. Not executing query...", err)
	}

	logger.InfoContext(ctx, fmt.Sprintf("Looking for app: %s, version: %s", appName, appVersion))
	query := `
		SELECT name, install_location, version, publisher
		FROM programs
		WHERE
		LOWER(name) LIKE LOWER('%` + appName + `%')
	`
	// The catalog name can differ from the registry DisplayName (e.g. catalog
	// "Amazon Corretto 25" vs DisplayName "Amazon Corretto (x64)"). The
	// unique_identifier is the value that should match programs.name, so search
	// on it as well.
	if uniqueIdentifier != "" && uniqueIdentifier != appName {
		if err := validateSqlInput(uniqueIdentifier); err != nil {
			return false, fmt.Errorf("Invalid character found in uniqueIdentifier: '%w'. Not executing query...", err)
		}
		query += `	OR LOWER(name) LIKE LOWER('%` + uniqueIdentifier + `%')`
	}
	if appPath != "" {
		query += fmt.Sprintf(" OR install_location LIKE '%%%s%%'", appPath)
	}
	cmd := exec.CommandContext(execTimeout, "osqueryi", "--json", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("osquery output: %s", string(output)))
		return false, fmt.Errorf("executing osquery command: %w", err)
	}

	type AppResult struct {
		Name            string `json:"name"`
		InstallLocation string `json:"install_location"`
		Version         string `json:"version"`
		Publisher       string `json:"publisher"`
	}
	var results []AppResult
	if err := json.Unmarshal(output, &results); err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("osquery output: %s", string(output)))
		return false, fmt.Errorf("parsing osquery JSON output: %w", err)
	}

	if len(results) > 0 {
		for _, result := range results {
			// Vendor is populated so name/version sanitizers that key off the
			// publisher (e.g. JetBrains build-number normalization in
			// MutateSoftwareOnIngestion) behave as they do in production.
			software := &fleet.Software{
				Name:    result.Name,
				Version: result.Version,
				Source:  "programs",
				Vendor:  result.Publisher,
			}
			queries.MutateSoftwareOnIngestion(ctx, software, logger)
			result.Version = software.Version
			result.Name = software.Name

			logger.InfoContext(ctx, fmt.Sprintf("Found app: '%s' at %s, Version: %s", result.Name, result.InstallLocation, result.Version))

			// Sublime Text's Inno Setup installer may not write version to registry properly
			// If app is found but version is empty, check if it's Sublime Text and skip version check
			if appName == "Sublime Text" && result.Version == "" {
				logger.InfoContext(ctx, "Sublime Text detected with empty version - skipping version check (installer may not write version to registry)")
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

			// Google Chrome auto-updates immediately after installation, so the
			// installed version may be newer than the installer version. If
			// version didn't match above, fall back to existence-only check.
			if appName == "Google Chrome" {
				logger.InfoContext(ctx, "Google Chrome detected - version mismatch but app is installed, skipping version check due to auto-update behavior")
				return true, nil
			}
			// Microsoft Office is a Click-to-Run product: the bootstrap setup.exe
			// always pulls the latest channel build from Microsoft's CDN, so the
			// installed version will typically be newer than the manifest version.
			// Only exempt genuine Office products (e.g. "Microsoft 365 Apps for
			// enterprise" or the older "Microsoft Office 365 ProPlus") — the broad
			// LIKE '%Microsoft Office%' search query also matches unrelated
			// Office-branded dependencies like "Open XML SDK 2.5 for Microsoft
			// Office" that must not prevent the post-uninstall check from reporting
			// the app as removed. The "Microsoft 365 Apps" prefix is used (not bare
			// "Microsoft 365") so Store apps such as "Microsoft 365 Copilot" aren't
			// treated as the Office suite. The publisher guard mirrors the
			// manifest's exists/patched queries (publisher = 'Microsoft
			// Corporation'), so a third-party app that happens to match a name
			// prefix can't bypass the version check.
			if appName == "Microsoft Office" &&
				result.Publisher == "Microsoft Corporation" &&
				(strings.HasPrefix(result.Name, "Microsoft 365 Apps") ||
					strings.HasPrefix(result.Name, "Microsoft Office")) {
				logger.InfoContext(ctx, "Microsoft Office detected - version mismatch but app is installed, skipping version check due to Click-to-Run always installing the latest build")
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
			logger.InfoContext(ctx, fmt.Sprintf("Found provisioned AppX package: '%s', Version: %s", provisioned.DisplayName, provisionedVersion))

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

	// OpenAI Codex CLI is a portable zip: it does not register in programs. Detect the binary via osquery file + PE version.
	if uniqueIdentifier == "Codex CLI" {
		ok, err := codexCLIExistsFromFile(execTimeout, logger, appVersion, appPath)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

func codexCLIExistsFromFile(ctx context.Context, logger *slog.Logger, appVersion, appPath string) (bool, error) {
	candidates := make([]string, 0, 3)
	if appPath != "" {
		candidates = append(candidates, filepath.Join(appPath, "codex.exe"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "Codex CLI", "codex.exe"))
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		candidates = append(candidates, filepath.Join(la, "Programs", "Codex CLI", "codex.exe"))
	}

	seen := make(map[string]struct{})
	for _, exePath := range candidates {
		if _, dup := seen[exePath]; dup {
			continue
		}
		seen[exePath] = struct{}{}

		if err := validateSqlInput(exePath); err != nil {
			continue
		}

		escaped := strings.ReplaceAll(exePath, "'", "''")
		query := `SELECT file_version FROM file WHERE path = '` + escaped + `'`
		cmd := exec.CommandContext(ctx, "osqueryi", "--json", query)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("osquery output: %s", string(output)))
			return false, fmt.Errorf("executing osquery file lookup: %w", err)
		}

		type fileResult struct {
			FileVersion string `json:"file_version"`
		}
		var results []fileResult
		if err := json.Unmarshal(output, &results); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("osquery output: %s", string(output)))
			return false, fmt.Errorf("parsing osquery JSON output: %w", err)
		}
		if len(results) == 0 || results[0].FileVersion == "" {
			continue
		}

		fileVer := results[0].FileVersion
		logger.InfoContext(ctx, fmt.Sprintf("Found Codex CLI binary at %s, file version: %s", exePath, fileVer))

		if fileVer == appVersion ||
			strings.HasPrefix(fileVer, appVersion+".") ||
			strings.HasPrefix(appVersion, fileVer+".") {
			return true, nil
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

	// Some installers (e.g. Visual Studio bootstrappers like vs_SSMS.exe)
	// download a large payload at install time and legitimately take longer
	// than a few minutes. Production allows up to 1 hour
	// (pkgscripts.MaxHostSoftwareInstallExecutionTime); 10 minutes is a
	// reasonable validator cap that covers large-payload installers without
	// letting a hung script run indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
