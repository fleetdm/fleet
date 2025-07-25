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
	query := `
		SELECT name, path, bundle_short_version, bundle_version 
		FROM apps
		WHERE 
		bundle_identifier LIKE '%` + uniqueAppIdentifier + `%' OR
		LOWER(name) LIKE LOWER('%` + appName + `%')
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
			level.Info(logger).Log("msg", fmt.Sprintf("Found app: '%s' at %s, Version: %s, Bundled Version: %s", result.Name, result.Path, result.Version, result.BundledVersion))
			if result.Version == appVersion || result.BundledVersion == appVersion {
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
