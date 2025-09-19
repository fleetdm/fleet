//go:build darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
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
			queries.MutateSoftwareOnIngestion(software, logger)
			result.Version = software.Version
			result.Name = software.Name

			level.Info(logger).Log("msg", fmt.Sprintf("Found app: '%s' at %s, Version: %s, Bundled Version: %s", result.Name, result.Path, result.Version, result.BundledVersion))
			if result.Version == appVersion || result.BundledVersion == appVersion {
				return true, nil
			}
		}
	}

	return false, nil
}

// executeScript writes `scriptContents` to a temp file and runs it with a timeout.
// It kills the entire process group on timeout and returns clear diagnostics.
func executeScript(cfg *Config, scriptContents string) (string, error) {
	// 1) Write the script file
	scriptPath := filepath.Join(cfg.tmpDir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing script: %w", err)
	}

	// 2) Set timeout
	to := 10 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	// 3) Prepare the command
	// Use /bin/sh to avoid relying on a shebang in the contents.
	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)

	// Ensure we can terminate the whole subtree (Adobe uninstallers spawn children).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Pass environment if provided.
	if len(cfg.env) > 0 {
		cmd.Env = append(os.Environ(), cfg.env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Start(); err != nil {
		// Start failures don't have pid or wait status.
		return "", fmt.Errorf("starting script: %w", err)
	}
	pid := cmd.Process.Pid

	err := cmd.Wait()
	dur := time.Since(start)

	// Build the result body (what you already log)
	outStr := stdout.String() + stderr.String()
	result := fmt.Sprintf(`
--------------------
%s
--------------------`, outStr)

	// 4) Timeout: kill the entire process group and return a clear error
	if ctx.Err() == context.DeadlineExceeded {
		// Negative pid targets the process group created by Setpgid.
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		return result, fmt.Errorf("script timed out after %s (pid=%d)", dur, pid)
	}

	// 5) Non-zero exit or killed by a signal: surface cause + output
	if err != nil {
		// Try to extract wait status details.
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
				if ws.Signaled() {
					return result, fmt.Errorf(
						"script killed by %s after %s (pid=%d)",
						ws.Signal(), dur, pid,
					)
				}
				return result, fmt.Errorf(
					"script failed with exit code %d after %s (pid=%d)",
					ws.ExitStatus(), dur, pid,
				)
			}
		}
		// Fallback if we can't decode status
		return result, fmt.Errorf("script failed after %s (pid=%d): %w", dur, pid, err)
	}

	// 6) Success
	return result, nil
}
