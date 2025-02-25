//go:build !windows
// +build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

// ChmodRestrictFile sets the appropriate permissions on a file so it can not be read by everyone
// On POSIX this is a normal chmod call.
func ChmodRestrictFile(path string) error {
	if err := os.Chmod(path, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("chmod restrict file: %w", err)
	}
	return nil
}

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

// SignalProcessBeforeTerminate just force terminate the target process
// Signaling the child process before termination is not supported on non-windows OSes
func SignalProcessBeforeTerminate(processName string) error {
	if processName == "" {
		return errors.New("processName should not be empty")
	}

	if err := killProcessByName(constant.DesktopAppExecName); err != nil && !errors.Is(err, ErrProcessNotFound) {
		return fmt.Errorf("There was an error kill target process %s: %w", processName, err)
	}

	return nil
}

// GetProcessesByName gets all running processes by its name.
// Returns ErrProcessNotFound if the process was not found running.
func GetProcessesByName(name string) ([]*gopsutil_process.Process, error) {
	if name == "" {
		return nil, errors.New("process name should not be empty")
	}

	processes, err := gopsutil_process.Processes()
	if err != nil {
		return nil, err
	}

	var foundProcesses []*gopsutil_process.Process
	for _, process := range processes {
		processName, err := process.Name()
		if err != nil {
			// No need to print errors here as this method might file for system processes
			continue
		}

		if strings.HasPrefix(processName, name) {
			foundProcesses = append(foundProcesses, process)
			break
		}
	}

	if len(foundProcesses) == 0 {
		return nil, ErrProcessNotFound
	}

	return foundProcesses, nil
}

func GetSMBiosUUID() (string, UUIDSource, error) {
	return "", UUIDSourceInvalid, errors.New("not implemented.")
}

// RunUpdateQuirks is a no-op on non-windows platforms
func PreUpdateQuirks() {
}

// IsInvalidReparsePoint is a no-op on non-windows platforms
func IsInvalidReparsePoint(err error) bool {
	return false
}
