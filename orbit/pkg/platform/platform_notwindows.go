//go:build !windows
// +build !windows

package platform

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

// ChmodExecutableDirectory sets the appropriate permissions on an executable
// file. On POSIX this is a normal chmod call.
func ChmodExecutableDirectory(path string) error {
	if err := os.Chmod(path, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("chmod executable directory: %w", err)
	}
	return nil
}

// ChmodExecutable sets the appropriate permissions on the parent directory of
// an executable file. On POSIX this is a regular chmod call.
func ChmodExecutable(path string) error {
	if err := os.Chmod(path, constant.DefaultExecutableMode); err != nil {
		return fmt.Errorf("chmod executable: %w", err)
	}
	return nil
}

// GracefulProcessKillByName looks for all the process running under a given name
// and force terminate them sending the SIGKILL signal
func GracefulProcessKillByName(name string) error {
	if name == "" {
		return errors.New("process name should not be empty")
	}

	// Getting the target process to gracefully shutdown
	foundProcess, _ := GetProcessByName(name)
	if foundProcess == nil {
		return nil // not an error, just no processes found
	}

	// Checking if target process is running
	// and force kill it if this happens to be the case
	isRunning, err := foundProcess.IsRunning()
	if err != nil {
		return fmt.Errorf("couldn't get running information on process %d: %w", foundProcess.Pid, err)
	}

	// Process is still running - force killing it
	if isRunning {
		if err := foundProcess.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
		}
	}

	return nil
}
