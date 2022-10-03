package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

var (
	ErrProcessNotFound    = errors.New("process not found")
	ErrComChannelNotFound = errors.New("comm channel not found")
)

// readPidFromFile reads a PID from a file
func readPidFromFile(destDir string, destFile string) (int32, error) {
	// Defense programming - sanity checks on inputs
	if destDir == "" {
		return 0, errors.New(" destination directory should not be empty")
	}

	if destFile == "" {
		return 0, errors.New(" destination file should not be empty")
	}

	pidFilePath := filepath.Join(destDir, destFile)
	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, fmt.Errorf("error reading pidfile %s: %w", pidFilePath, err)
	}

	intNumber, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error converting pidfile %s: %w", pidFilePath, err)
	}

	return int32(intNumber), err
}

// processNameMatches returns whether the process running with the given pid matches
// the executable name (case insensitive).
// If there's no process running with the given pid then (false, nil) is returned.
func processNameMatches(pid int, expectedPrefix string) (bool, error) {
	if pid == 0 {
		return false, errors.New("process id should not be zero")
	}

	if expectedPrefix == "" {
		return false, errors.New("expected prefix should not be empty")
	}

	process, err := ps.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("find process: %d: %w", pid, err)
	}

	if process == nil {
		return false, nil
	}

	return strings.HasPrefix(strings.ToLower(process.Executable()), strings.ToLower(expectedPrefix)), nil
}

// killPID kills a process by PID
func killPID(pid int32) error {
	if pid == 0 {
		return errors.New("process id should not be zero")
	}

	processes, err := gopsutil_process.Processes()
	if err != nil {
		return err
	}

	for _, process := range processes {
		if pid == process.Pid {
			process.Kill()
			break
		}
	}

	return nil
}

// KillProcessByName kills a single process by its name
func KillProcessByName(name string) error {
	if name == "" {
		return errors.New("process name should not be empty")
	}

	foundProcess, err := GetProcessByName(name)
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	if err := foundProcess.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
	}

	return nil
}

// getProcessesByName gets a single process object by its name
func getProcessesByName(name string) ([]*gopsutil_process.Process, error) {
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
		}
	}

	if len(foundProcesses) == 0 {
		return nil, ErrProcessNotFound
	}

	return foundProcesses, nil
}

// KillAllProcessByName kills all process found by their name
func KillAllProcessByName(name string) error {
	if name == "" {
		return errors.New("process name should not be empty")
	}

	foundProcesses, err := getProcessesByName(name)
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	// Killing found processes
	for _, foundProcess := range foundProcesses {
		if err := foundProcess.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
		}
	}

	return nil
}

// KillFromPIDFile kills a process taking the PID value from a file
func KillFromPIDFile(destDir string, pidFileName string, expectedExecName string) error {
	if destDir == "" {
		return errors.New("destination directory should not be empty")
	}

	if pidFileName == "" {
		return errors.New("PID file name should not be empty")
	}

	if expectedExecName == "" {
		return errors.New("expected executable name should not be empty")
	}

	pid, err := readPidFromFile(destDir, pidFileName)
	switch {
	case err == nil:
		// OK
	case errors.Is(err, os.ErrNotExist):
		return nil // we assume it's not running
	default:
		return fmt.Errorf("reading pid from: %s: %w", destDir, err)
	}

	matches, err := processNameMatches(int(pid), expectedExecName)
	if err != nil {
		return fmt.Errorf("inspecting process %d: %w", pid, err)
	}

	if !matches {
		// Nothing to do, another process may be running with this pid
		// (e.g. could happen after a restart).
		return nil
	}

	if err := killPID(pid); err != nil {
		return fmt.Errorf("killing %d: %w", pid, err)
	}

	return nil
}
