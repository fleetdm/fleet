package platform

import (
	"errors"
	"fmt"
	"strings"

	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

var (
	ErrProcessNotFound    = errors.New("process not found")
	ErrComChannelNotFound = errors.New("comm channel not found")
)

type UUIDSource string

const (
	UUIDSourceInvalid  = "UUID_Source_Invalid"
	UUIDSourceWMI      = "UUID_Source_WMI"
	UUIDSourceHardware = "UUID_Source_Hardware"
)

// killProcessByName kills a single process by its name.
func killProcessByName(name string) error {
	if name == "" {
		return errors.New("process name should not be empty")
	}

	foundProcesses, err := GetProcessesByName(name)
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	for _, foundProcess := range foundProcesses {
		if err := foundProcess.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
		}
	}

	return nil
}

// getProcessesByName returns all the running processes with the given prefix in their name.
func getProcessesByName(namePrefix string) ([]*gopsutil_process.Process, error) {
	if namePrefix == "" {
		return nil, errors.New("process name prefix should not be empty")
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

		if strings.HasPrefix(processName, namePrefix) {
			foundProcesses = append(foundProcesses, process)
		}
	}

	return foundProcesses, nil
}

// Process holds basic information of a process.
type Process struct {
	// Name is the name of the process.
	Name string
	// PID is the process identifier.
	PID int32
}

// KillAllProcessByName kills all the running processes with the given prefix in their name.
// It returns the processes that were killed. It returns `nil, nil` if there were no processes
// running with such name prefix.
func KillAllProcessByName(namePrefix string) ([]Process, error) {
	if namePrefix == "" {
		return nil, errors.New("process name prefix should not be empty")
	}

	foundProcesses, err := getProcessesByName(namePrefix)
	if err != nil {
		return nil, fmt.Errorf("get processes by name: %w", err)
	}

	var killedProcesses []Process
	for _, foundProcess := range foundProcesses {
		processName, _ := foundProcess.Name()
		if err := foundProcess.Kill(); err != nil {
			return nil, fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
		}
		killedProcesses = append(killedProcesses, Process{
			Name: processName,
			PID:  foundProcess.Pid,
		})
	}

	return killedProcesses, nil
}
