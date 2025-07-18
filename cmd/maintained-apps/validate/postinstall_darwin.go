//go:build darwin

package main

import (
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
