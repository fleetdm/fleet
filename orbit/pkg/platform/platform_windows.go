//go:build windows
// +build windows

package platform

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"

	"github.com/digitalocean/go-smbios/smbios"
	"github.com/hectane/go-acl"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"
)

const (
	fullControl    = uint32(2032127)
	readAndExecute = uint32(131241)
)

// ChmodRestrictFile sets the appropriate permissions on a file so it can not be read by everyone
// On POSIX this is a normal chmod call.
func ChmodRestrictFile(path string) error {
	if err := acl.Apply(
		path,
		true,
		false,
		acl.GrantSid(windows.GENERIC_ALL, constant.SystemSID),
		acl.GrantSid(windows.GENERIC_ALL, constant.AdminSID),
		acl.GrantSid(0, constant.UserSID), // no access permissions for regular users
	); err != nil {
		return fmt.Errorf("restricting file access: %w", err)
	}

	return nil
}

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

	// Closing the handle to avoid handle leaks.
	defer windows.CloseHandle(h) //nolint:errcheck

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
	// Closing the handle to avoid handle leaks.
	defer windows.CloseHandle(snapshot) //nolint:errcheck

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

// It obtains the BIOS UUID by calling "cmd.exe /c wmic csproduct get UUID" and parsing the results
func wmiGetSMBiosUUID() (string, error) {
	args := []string{"/C", "wmic csproduct get UUID"}
	out, err := exec.Command("cmd", args...).Output()
	if err != nil {
		return "", err
	}
	uuidOutputStr := string(out)
	if len(uuidOutputStr) == 0 {
		return "", errors.New("get UUID: output from wmi is empty")
	}
	outputByLines := strings.Split(strings.TrimRight(uuidOutputStr, "\n"), "\n")
	if len(outputByLines) < 2 {
		return "", errors.New("get UUID: unexpected output")
	}
	return strings.TrimSpace(outputByLines[1]), nil
}

// It performs a UUID sanity check on a given byte array
// The sectionPayloadBytes buffer contains the Smbios Structure Type 1 payload - This includes the actual UUID bytes + Optional section strings
func isValidUUID(sectionPayloadBytes []byte) (bool, error) {
	// SMBIOS constants from spec here - https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.1.1.pdf
	const uuidSize int = 0x10 // UUID size is calculated with field offset value (0xA) + node field length (6 bytes) - 16 bytes - 128bits long

	// Sanity check on min size of the input buffer
	// Buffer should be long enough to contain an UUID
	if len(sectionPayloadBytes) < uuidSize {
		return false, errors.New("Invalid input UUID size")
	}

	// UUID field sanity check for null values
	// Logic is based on https://github.com/ContinuumLLC/godep-go-smbios/blob/ab7c733f1be8e55ed3e0587d1aa2d5883fe8801e/smbios/decoder.go#L135
	only0xFF, only0x00 := true, true
	for i := 0; i < uuidSize && (only0x00 || only0xFF); i++ {
		if sectionPayloadBytes[i] != 0x00 {
			only0x00 = false
		}
		if sectionPayloadBytes[i] != 0xFF {
			only0xFF = false
		}
	}

	if only0xFF {
		return false, errors.New("UUID is not currently present in the system, but it can be set.")
	}
	if only0x00 {
		return false, errors.New("UUID is not present in the system.")
	}

	return true, nil
}

// It obtains the BIOS UUID value by reading the SMBIOS "System Information"
// structure data on the OS SMBIOS interface.
// On Windows, the SMBIOS "System Information" data can be obtained by calling GetSystemFirmwareTable()
// https://learn.microsoft.com/en-us/windows/win32/api/sysinfoapi/nf-sysinfoapi-getsystemfirmwaretable
// Instead of just calling this native API, this function relies on Digital Ocean's go-smbios
// library. This package smbios provides detection and access to System Management BIOS (SMBIOS) and
// Desktop Management Interface (DMI) data and structures across: https://github.com/digitalocean/go-smbios
// This function should work as is on Linux thanks to the go-smbios interface abstraction. See the
// list of supported OSes on the go-smbios documentation.
// The windows go-smbios implementations calls to GetSystemFirmwareTable()
func hardwareGetSMBiosUUID() (string, error) {
	// SMBIOS data in operating system-specific location
	streamReader, smBIOSRawData, err := smbios.Stream()
	if err != nil {
		return "", fmt.Errorf("failed to open stream: %v", err)
	}

	// Ensure that stream will be closed
	defer streamReader.Close()

	// Decode SMBIOS structures from the stream.
	decoder := smbios.NewDecoder(streamReader)
	structSMBIOSdata, err := decoder.Decode()
	if err != nil {
		return "", fmt.Errorf("failed to decode BIOS structures: %v", err)
	}

	// Determine SMBIOS version and table location from entry point
	biosMajor, biosMinor, _ := smBIOSRawData.Version()

	// SMBIOS constants from spec here - https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.1.1.pdf
	const systemInformationType uint8 = 0x01 // System Information indicator
	const minBiosStructSize uint8 = 0x1b     // Section 7.2 on the SMBIOS specification (0x1a min header size + null character)
	const uuidOffset uint8 = 0x4             // UUID offset in System Information (Type 1) structure
	const revMajorVersion int = 0x3          // SMBIOS revision that most of the current BIOS have - v3 specs were released in 2015
	const minLegacyMajorVersion int = 0x2    // Minimum SMBIOS Major rev that supports UUID little-endian encoding
	const minLegacyMinorVersion int = 0x6    // Minimum SMBIOS Minor rev that supports UUID little-endian encoding

	// Walking the obtained SMBIOS data
	for _, rawBiosStruct := range structSMBIOSdata {
		if (rawBiosStruct.Header.Type == systemInformationType) && (rawBiosStruct.Header.Length >= minBiosStructSize) {
			uuidBytes := rawBiosStruct.Formatted[uuidOffset:]

			// UUID sanity check
			isValidUUID, err := isValidUUID(uuidBytes)
			if err != nil {
				return "", fmt.Errorf("%v", err)
			}

			if !isValidUUID {
				return "", errors.New("invalid UUID")
			}

			// As of version 2.6 of the SMBIOS specification, the first 3 fields of the UUID are
			// supposed to be encoded in little-endian (section 7.2.1)
			var smBiosUUID string = ""
			if (biosMajor >= revMajorVersion) || (biosMajor >= minLegacyMajorVersion && biosMinor >= minLegacyMinorVersion) {
				smBiosUUID = fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
					uuidBytes[3], uuidBytes[2], uuidBytes[1], uuidBytes[0], uuidBytes[5], uuidBytes[4], uuidBytes[7], uuidBytes[6], uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11], uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
			} else {
				smBiosUUID = fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
					uuidBytes[0], uuidBytes[1], uuidBytes[2], uuidBytes[3], uuidBytes[4], uuidBytes[5], uuidBytes[6], uuidBytes[7], uuidBytes[8], uuidBytes[9], uuidBytes[10], uuidBytes[11], uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])
			}

			return smBiosUUID, nil
		}
	}

	return "", errors.New("UUID was not found")
}

// It attempts to get SMBIOS UUID through WMI, and if this mechanism fails, it fallback into reading
// the actual SMBIOS hardware interface.
func GetSMBiosUUID() (string, UUIDSource, error) {
	// It attempts first to get the UUID from WMI
	uuid, err := wmiGetSMBiosUUID()
	if err != nil {

		// If WMI fails, it fallback into reading the SMBIOS HW interface
		uuid, err := hardwareGetSMBiosUUID()
		if err != nil {
			return "", "", fmt.Errorf("UUID could not be obtained through WMI and Hardware routes: %w", err)
		}

		// UUID was obtained from reading the hardware SMBIOS UUID data
		return uuid, UUIDSourceHardware, nil
	}

	// UUID was obtained from calling WMI infrastructure
	return uuid, UUIDSourceWMI, nil
}
