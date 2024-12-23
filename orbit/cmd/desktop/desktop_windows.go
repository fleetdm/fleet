//go:build windows
// +build windows

package main

import (
	_ "embed"
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

// For Windows we must use ico format for the icon,
// see https://github.com/getlantern/systray/blob/6065fda28be8c8d91aeb5e20de25e1600b8664a3/systray_windows.go#L850-L856.

//go:embed windows_app.ico
var iconLight []byte

//go:embed windows_app.ico
var iconDark []byte

// MessageBox displays a Windows message box with the specified title and text.
func MessageBox(title, text string) {
	// Load user32.dll
	user32 := syscall.MustLoadDLL("user32.dll")
	defer user32.Release()

	// Load MessageBoxW function
	messageBox := user32.MustFindProc("MessageBoxW")

	// Convert strings to UTF16
	textUTF16, _ := syscall.UTF16PtrFromString(text)
	titleUTF16, _ := syscall.UTF16PtrFromString(title)

	// Call MessageBoxW
	messageBox.Call(
		0,                                   // Handle to owner window (0 for no owner)
		uintptr(unsafe.Pointer(textUTF16)),  // Text message
		uintptr(unsafe.Pointer(titleUTF16)), // Title
		0,                                   // Style (0 for OK button)
	)
}

// blockWaitForStopEvent waits for the named event kernel object to be signalled
func blockWaitForStopEvent(channelId string) error {
	if channelId == "" {
		return errors.New("communication channel name should not be empty")
	}

	// converting go string to UTF16 windows compatible string
	targetChannel := "Global\\comm-" + channelId
	ev, err := windows.UTF16PtrFromString(targetChannel)
	if err != nil {
		return fmt.Errorf("there was a problem generating UTF16 string: %w", err)
	}

	// The right to use the object for synchronization
	// https://learn.microsoft.com/en-us/windows/win32/sync/synchronization-object-security-and-access-rights
	const EVENT_SYNCHRONIZE = 0x00100000

	// block wait until channel is available
	var handle windows.Handle = windows.InvalidHandle
	for {
		// OpenEvent API opens a named event object from the kernel object manager
		// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-openeventw
		handle, err = windows.OpenEvent(EVENT_SYNCHRONIZE, false, ev)
		if (err == nil) && (handle != windows.InvalidHandle) {
			break
		}

		// wait before next handle check
		time.Sleep(500 * time.Millisecond)
	}

	defer windows.CloseHandle(handle)

	// OpenEvent() call was successful and our process got a handle to the named event kernel object
	log.Info().Msg("Comm channel was acquired")

	// now block wait for the handle to be signaled by Orbit
	// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-waitforsingleobject
	s, err := windows.WaitForSingleObject(handle, windows.INFINITE)
	if (err != nil) && (err != windows.ERROR_SUCCESS) {
		return fmt.Errorf("there was a problem calling WaitForSingleObject: %w", err)
	}

	if s != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("event wait was interrupted for unknown reasons: %d", s)
	}

	// Display a confirmation message after successful installation
	MessageBox("Installation Successful", "The Fleet agent has been installed and enrolled successfully!")

	return nil
}
