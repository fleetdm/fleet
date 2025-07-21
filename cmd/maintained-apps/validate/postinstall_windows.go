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
)

var preInstalled = []string{
	"google-chrome/windows",
}

func postApplicationInstall(_ string) error {
	return nil
}

func doesAppExists(appName, _ string, appVersion, appPath string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Looking for app: %s, version: %s\n", appName, appVersion)
	query := `
		SELECT name, install_location, version 
		FROM programs
		WHERE
		LOWER(name) LIKE LOWER('%` + appName + `%')
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

func executeScript(scriptContents string) (string, error) {
	scriptExtension := ".ps1"
	scriptPath := filepath.Join(tmpDir, "script"+scriptExtension)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing script: %w", err)
	}

	// Use custom execution with non-interactive flags for Windows
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.WaitDelay = 1 * time.Minute
	cmd.Env = env
	cmd.Dir = filepath.Dir(scriptPath)

	output, err := cmd.CombinedOutput()

	exitCode := -1

	// Only set exitCode if process completed and context wasn't cancelled
	if cmd.ProcessState != nil {
		exitCode = int(int32(cmd.ProcessState.ExitCode()))
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
