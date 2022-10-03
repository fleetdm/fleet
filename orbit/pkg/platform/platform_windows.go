//go:build windows
// +build windows

package platform

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"

	"github.com/hectane/go-acl"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"
)

const (
	fullControl    = uint32(2032127)
	readAndExecute = uint32(131241)
)

// ChmodExecutableDirectory sets the appropriate permissions on the parent
// directory of an executable file. On Windows this involves setting the
// appropriate ACLs.
func ChmodExecutableDirectory(path string) error {
	if err := acl.Apply(
		path,
		true,
		false,
		acl.GrantSid(fullControl, constant.SystemSID),
		acl.GrantSid(fullControl, constant.AdminSID),
		acl.GrantSid(readAndExecute, constant.UserSID),
	); err != nil {
		return fmt.Errorf("apply ACLs: %w", err)
	}

	return nil
}

// ChmodExecutable sets the appropriate permissions on an executable file. On
// Windows this involves setting the appropriate ACLs.
func ChmodExecutable(path string) error {
	if err := acl.Apply(
		path,
		true,
		false,
		acl.GrantSid(fullControl, constant.SystemSID),
		acl.GrantSid(fullControl, constant.AdminSID),
		acl.GrantSid(readAndExecute, constant.UserSID),
	); err != nil {
		return fmt.Errorf("apply ACLs: %w", err)
	}

	return nil
}

// signalThroughNamedEvent signals a target named event kernel object
func signalThroughNamedEvent(channelId string) error {
	if channelId == "" {
		return errors.New("communication channel name should not be empty")
	}

	// converting go string to UTF16 windows compatible string
	targetChannel := "Global\\comm-" + channelId
	ev, err := windows.UTF16PtrFromString(targetChannel)
	if err != nil {
		return fmt.Errorf("there was a problem generating UTF16 string: %w", err)
	}

	// OpenEvent Api opens a named event object from the kernel object manager
	// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-openeventw
	h, err := windows.OpenEvent(windows.EVENT_ALL_ACCESS, false, ev)
	if (err != nil) && (err != windows.ERROR_SUCCESS) {
		return fmt.Errorf("there was a problem calling OpenEvent: %w", err)
	}

	if h == windows.InvalidHandle {
		return errors.New("event handle is invalid")
	}

	defer windows.CloseHandle(h) // closing the handle to avoid handle leaks

	// signaling the event
	// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-setevent
	err = windows.PulseEvent(h)
	if (err != nil) && (err != windows.ERROR_SUCCESS) {
		return fmt.Errorf("there was an issue signaling the event: %w", err)
	}

	// Dumb sleep to ensure the remote process to pick up the windows message
	time.Sleep(500 * time.Millisecond)

	return nil
}

// SignalProcessBeforeTerminate signals a named event kernel object
// before force terminate a process
func SignalProcessBeforeTerminate(processName string) error {
	if processName == "" {
		return errors.New("processName should not be empty")
	}

	if err := signalThroughNamedEvent(processName); err != nil {
		return ErrComChannelNotFound
	}

	foundProcess, err := GetProcessByName(processName)
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	isRunning, err := foundProcess.IsRunning()
	if (err == nil) && (isRunning) {
		if err := foundProcess.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
		}
	}
	return nil
}

// GetProcessByName gets a single process object by its name
func GetProcessByName(name string) (*gopsutil_process.Process, error) {
	if name == "" {
		return nil, errors.New("process name should not be empty")
	}

	// We gather information around running processes on the system
	// CreateToolhelp32Snapshot() is used for this
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-createtoolhelp32snapshot
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}

	// sanity check on returned snapshot handle
	if snapshot == windows.InvalidHandle {
		return nil, errors.New("the snapshot returned returned by CreateToolhelp32Snapshot is invalid")
	}
	defer windows.CloseHandle(snapshot)

	var foundProcessID uint32 = 0

	// Initializing work structure PROCESSENTRY32W
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/ns-tlhelp32-processentry32w
	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))

	// And finally iterating the snapshot by calling Process32First()
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-process32first
	if err := windows.Process32First(snapshot, &procEntry); err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}

	// Process32First() is going to return ERROR_NO_MORE_FILES when no more threads present
	// it will return FALSE/nil otherwise
	for err == nil {

		if strings.HasPrefix(syscall.UTF16ToString(procEntry.ExeFile[:]), name) {
			foundProcessID = procEntry.ProcessID
			break
		}

		// Process32Next() is calling to keep iterating the snapshot
		// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-process32next
		err = windows.Process32Next(snapshot, &procEntry)
	}

	process, err := gopsutil_process.NewProcess(int32(foundProcessID))
	if err != nil {
		return nil, fmt.Errorf("NewProcess: %w", err)
	}

	return process, nil
}
