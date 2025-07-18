//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func postApplicationInstall(appPath string) error {
	err := forceLaunchServicesRefresh(appPath)
	if err != nil {
		return fmt.Errorf("Error forcing LaunchServices refresh: %v. Attempting to continue", err)
	}
	err = removeAppQuarentine(appPath)
	if err != nil {
		return fmt.Errorf("Error removing app quarantine: %v. Attempting to continue", err)
	}
	return nil
}

func removeAppQuarentine(appPath string) error {
	if appPath == "" {
		return nil
	}
	fmt.Printf("Attempting to remove quarantine for: '%s'\n", appPath)
	cmd := exec.Command("xattr", "-p", "com.apple.quarantine", appPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("checking quarantine status: %v\n", err)
	}
	fmt.Printf("Quarantine status: '%s'\n", strings.TrimSpace(string(output)))
	cmd = exec.Command("spctl", "-a", "-v", appPath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("checking spctl status: %v\n", err)
	}
	fmt.Printf("spctl status: '%s'\n", strings.TrimSpace(string(output)))

	cmd = exec.Command("sudo", "spctl", "--add", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("adding app to quarantine exceptions: %w", err)
	}

	cmd = exec.Command("sudo", "xattr", "-r", "-d", "com.apple.quarantine", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("removing quarantine attribute: %w", err)
	}

	return nil
}

func forceLaunchServicesRefresh(appPath string) error {
	if appPath == "" {
		return nil
	}
	fmt.Printf("Forcing LaunchServices refresh for: '%s'\n", appPath)
	cmd := exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-f", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forcing LaunchServices refresh: %w", err)
	}
	time.Sleep(2 * time.Second)
	return nil
}

func doesAppExists(appName, uniqueAppIdentifier, appVersion, appPath string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Looking for app: %s, version: %s\n", appName, appVersion)
	query := `
		SELECT name, path, bundle_short_version, bundle_version 
		FROM apps
		WHERE 
		bundle_identifier LIKE '%` + uniqueAppIdentifier + `%' OR
		name ILIKE '%` + appName + `%'
	`
	if appPath != "" {
		query += fmt.Sprintf(" OR path LIKE '%%%s%%'", appPath)
	}
	cmd := exec.CommandContext(ctx, "osqueryi", "--json", query)
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
			fmt.Printf("Found app: '%s' at %s, Version: %s, Bundled Version: %s\n", result.Name, result.Path, result.Version, result.BundledVersion)
			if result.Version == appVersion || result.BundledVersion == appVersion {
				return true, nil
			}
		}
	}

	return false, nil
}
