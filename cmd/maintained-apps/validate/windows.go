//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var preInstalled = []string{}

func postApplicationInstall(_ kitlog.Logger, _ string) error {
	return nil
}

func appExists(ctx context.Context, logger kitlog.Logger, appName, _, appVersion, appPath string) (bool, error) {
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
			level.Info(logger).Log("msg", fmt.Sprintf("Found app: '%s' at %s, Version: %s", result.Name, result.InstallLocation, result.Version))
			if result.Version == appVersion {
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

	// Use custom execution with non-interactive flags for Windows
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
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
