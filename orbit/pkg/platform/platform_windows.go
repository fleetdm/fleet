//go:build windows
// +build windows

package platform

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/hectane/go-acl"
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

// getThreadsByProcess returns the thread ids of the threads running on a given process
// The thread information is provided by CreateToolhelp32Snapshot()
func getThreadsByProcess(pid uint32) (*[]uint32, error) {
	var threadIDs []uint32

	// We gather information on the threads running on a given process first
	// CreateToolhelp32Snapshot() is used for this
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-createtoolhelp32snapshot
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, pid)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}

	// sanity check on returned snapshot handle
	if snapshot == windows.InvalidHandle {
		return nil, errors.New("the snapshot returned returned by CreateToolhelp32Snapshot is invalid")
	}
	defer windows.CloseHandle(snapshot)

	// Initializing work structure THREADENTRY32
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/ns-tlhelp32-threadentry32
	var thStruct windows.ThreadEntry32
	thStruct.Size = uint32(unsafe.Sizeof(thStruct))

	// And finally iterating the snapshot by calling Thread32First()
	// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-thread32first
	if err := windows.Thread32First(snapshot, &thStruct); err != nil {
		return nil, fmt.Errorf("Thread32First: %w", err)
	}

	// Thread32First() is going to return ERROR_NO_MORE_FILES when no more threads present
	// it will return FALSE/nil otherwise
	for err == nil {

		// Sanity check to ensure that only threads ids of the given process are saved
		if thStruct.OwnerProcessID == pid {
			threadIDs = append(threadIDs, thStruct.ThreadID)
		}

		// Thread32Next() is calling to keep iterating the snapshot
		// https://learn.microsoft.com/en-us/windows/win32/api/tlhelp32/nf-tlhelp32-thread32next
		err = windows.Thread32Next(snapshot, &thStruct)
	}

	if len(threadIDs) > 0 {
		return &threadIDs, nil
	} else {
		return &threadIDs, errors.New("No threads were found")
	}
}

// sendCloseWindowCmd sends the WINDOWS_CLOSE to windows running on a given thread
// Code based on https://github.com/golang/go/blob/master/src/runtime/syscall_windows_test.go#L138
func sendCloseWindowCmd(tid uint32) error {
	const (
		// Retcode that indicates a successfull IsWindow() call
		IW_Successful_Call = 1

		// Retcode that indicates the enumeration should continue
		CB_Continue_Enumeration = 1

		// Code that indicates an empty SendMessage() parameter
		SM_Empty_Param = 0

		// WM_Close message to be send to a given window object
		// This message signal the target window to terminate
		// https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-close
		SM_WM_CLOSE = 0x0010

		// Retcode that indicates a successfull EnumThreadWindows() call
		ET_Successful_Call = 1
	)

	var (
		// ensuring user32.dll is mapped to memory
		moduser32 = windows.NewLazySystemDLL("user32.dll")

		// ensuring function pointers to dll exports
		IsWindowsProc         = moduser32.NewProc("IsWindow")
		SendMessageProc       = moduser32.NewProc("SendMessageW")
		EnumThreadWindowsProc = moduser32.NewProc("EnumThreadWindows")
	)

	if (IsWindowsProc == nil) || (SendMessageProc == nil) || (EnumThreadWindowsProc == nil) {
		return errors.New("user32.dll exports cannot be found")
	}

	// Sending messages to the given thread id

	// lets's define first the EnumThreadWndProc callback function
	// https://learn.microsoft.com/en-us/previous-versions/windows/desktop/legacy/ms633496(v=vs.85)
	wndEnumCallback := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		// then check if hwnd is a valid windows handle
		retCode, _, err := IsWindowsProc.Call(uintptr(hwnd))
		if err != windows.ERROR_SUCCESS {
			return CB_Continue_Enumeration // check next available window object
		}

		if retCode == IW_Successful_Call {
			// We have a valid window object on this thread
			// SendMessage() is used to send the WINDOWS_CLOSE message
			// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendmessage

			// Best effort call - no need to log error or check result
			SendMessageProc.Call(uintptr(hwnd), SM_WM_CLOSE, SM_Empty_Param, SM_Empty_Param)
		}

		return CB_Continue_Enumeration // continue enumeration
	})

	// calling EnumThreadWindows with a callback to enumerate the threads of a given process
	// We need to check if the are windows associated to these threads and send
	// messages to gracefully close those windows
	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-enumthreadwindows
	retCode, _, err := EnumThreadWindowsProc.Call(uintptr(tid), wndEnumCallback, 0)
	if (err != windows.ERROR_SUCCESS) || (retCode != ET_Successful_Call) {
		return fmt.Errorf("there was a problem calling EnumThreadWindows(): (%d) %w", retCode, err)
	} else {
		return nil
	}
}

// GracefulProcessKillByName looks for all the process running under a given name
// and attempts to graceful kill the first by sending WM_CLOSE messages
// if this fails, processes are force terminated
func GracefulProcessKillByName(name string) error {
	if name == "" {
		return errors.New("process name should not be empty")
	}

	// Getting the target process to gracefully shutdown
	foundProcess, _ := GetProcessByName(name)
	if foundProcess == nil {
		return nil // not an error, just no processes found
	}

	// Grabbing the threads of the target process
	threadsIDs, err := getThreadsByProcess(uint32(foundProcess.Pid))
	if err != nil {
		return fmt.Errorf("get process threads: %w", err)
	}

	err = execuser.StartLoggedOnUserImpersonation()
	if err != nil {
		fmt.Printf("error starting impersonation: %v", err)
	}

	// the send the WINDOWS_CLOSE to windows objects
	// that might be running on these threads
	for _, threadID := range *threadsIDs {
		// best effort call - no logging involved
		sendCloseWindowCmd(threadID)
	}

	err = execuser.StopImpersonation()
	if err != nil {
		fmt.Printf("error starting impersonation: %v", err)
	}

	// The sent message should have been processed syncronously
	// by the target windows objects event's loop
	// Giving some extra time just in case
	time.Sleep(100 * time.Millisecond)

	// Now checking if target process is still running
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
