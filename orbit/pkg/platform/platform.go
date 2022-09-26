package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
	"github.com/rs/zerolog/log"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

var ErrProcessNotFound = errors.New("process not found")

// Kills a single process by its name
func KillProcessByName(name string) error {
	if name == "" {
		return fmt.Errorf("Invalid argument was provided")
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

// Gets a single process object by its name
func GetProcessByName(name string) (*gopsutil_process.Process, error) {
	if name == "" {
		return nil, fmt.Errorf("Invalid argument was provided")
	}

	processes, err := gopsutil_process.Processes()
	if err != nil {
		return nil, err
	}

	var foundProcess *gopsutil_process.Process
	for _, process := range processes {
		processName, err := process.Name()
		if err != nil {
			log.Debug().Err(err).Int32("pid", process.Pid).Msg("get process name")
			continue
		}

		if strings.HasPrefix(processName, name) {
			foundProcess = process
			break
		}
	}

	if foundProcess == nil {
		return nil, ErrProcessNotFound
	}

	return foundProcess, nil
}

// Reads a PID from a file
func ReadPidFromFile(destDir string, destFile string) (int, error) {
	if (destDir == "") || (destFile == "") {
		return 0, fmt.Errorf("Invalid arguments were provided")
	}

	pidFilePath := filepath.Join(destDir, destFile)
	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, fmt.Errorf("error reading pidfile %s: %w", pidFilePath, err)
	}

	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// ProcessNameMatches returns whether the process running with the given pid matches
// the executable name (case insensitive).
// If there's no process running with the given pid then (false, nil) is returned.
func ProcessNameMatches(pid int, expectedPrefix string) (bool, error) {
	if (pid == 0) || (expectedPrefix == "") {
		return false, fmt.Errorf("Invalid arguments were provided")
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

// Kills a process by PID
func KillPID(pid int32) error {
	if pid == 0 {
		return fmt.Errorf("Invalid arguments were provided")
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

// Kills a process taking the PID value from a file
func KillFromPIDFile(destDir string, pidFileName string, expectedExecName string) error {
	if (destDir == "") || (pidFileName == "") || (expectedExecName == "") {
		return fmt.Errorf("Invalid arguments were provided")
	}

	pid, err := ReadPidFromFile(destDir, pidFileName)
	switch {
	case err == nil:
		// OK
	case errors.Is(err, os.ErrNotExist):
		return nil // we assume it's not running
	default:
		return fmt.Errorf("reading pid from: %s: %w", destDir, err)
	}

	matches, err := ProcessNameMatches(pid, expectedExecName)
	if err != nil {
		return fmt.Errorf("inspecting process %d: %w", pid, err)
	}

	if !matches {
		// Nothing to do, another process may be running with this pid
		// (e.g. could happen after a restart).
		return nil
	}

	if err := KillPID(int32(pid)); err != nil {
		return fmt.Errorf("killing %d: %w", pid, err)
	}

	return nil
}
