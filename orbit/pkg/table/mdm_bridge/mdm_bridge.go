//go:build windows

//nolint:gosec // G103: unsafe required for Windows API calls.
package mdm_bridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/digitalocean/go-smbios/smbios"
	"github.com/hillu/go-ntdll"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"github.com/scjalliance/comshim"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	// Windows DLL and functions for runtime binding
	modmdmregistration = windows.NewLazySystemDLL("mdmregistration.dll")
	modmdmlocalmgmt    = windows.NewLazySystemDLL("mdmlocalmanagement.dll")
	modkernel32        = windows.NewLazySystemDLL("kernel32.dll")
	modembeddedmode    = windows.NewLazySystemDLL("embeddedmodesvcapi.dll")

	procIsDeviceRegisteredWithManagement  = modmdmregistration.NewProc("IsDeviceRegisteredWithManagement")
	procSendSyncMLcommand                 = modmdmlocalmgmt.NewProc("ApplyLocalManagementSyncML")
	procRegisterDeviceWithLocalManagement = modmdmlocalmgmt.NewProc("RegisterDeviceWithLocalManagement")
	proclstrlenW                          = modkernel32.NewProc("lstrlenW")
	procRtlMoveMemory                     = modkernel32.NewProc("RtlMoveMemory")
	procIsEmbeddedModeAllowed             = modembeddedmode.NewProc("IsEmbeddedModeAllowed")

	// Synchronization mutex
	mu sync.Mutex

	// MDM Management stack initialization executor
	mdmManagementStackInit sync.Once

	// Value written to the EmbeddedMode Flags registry entry to allow MDM command execution
	flagsValue     []byte
	flagsValueInit sync.Once
)

const (
	maxBufSize            = 2048 // Max Unicode Length for UPN - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-adls/63f5e067-d1b3-4e6e-9e53-a92953b6005b
	flagsRegistryLocation = `SYSTEM\CurrentControlSet\Services\embeddedmode\Parameters`
	flagsRegistryName     = `Flags`
	mdmMutexName          = "__MDM_LOCAL_MANAGEMENT_NAMED_MUTEX__"
)

// SyncML XML Parsing Types
type SyncMLHeader struct {
	DTD        string `xml:"VerDTD"`
	Version    string `xml:"VerProto"`
	SessionID  int    `xml:"SessionID"`
	MsgID      int    `xml:"MsgID"`
	Target     string `xml:"Target>LocURI"`
	Source     string `xml:"Source>LocURI"`
	MaxMsgSize int    `xml:"Meta>A:MaxMsgSize"`
}

type SyncMLCommandMeta struct {
	XMLinfo string `xml:"xmlns,attr"`
	Type    string `xml:"Type"`
}

type SyncMLCommandItem struct {
	Meta   SyncMLCommandMeta `xml:"Meta"`
	Source string            `xml:"Source>LocURI"`
	Data   string            `xml:"Data"`
}

type SyncMLCommand struct {
	XMLName xml.Name
	CmdID   int                 `xml:",omitempty"`
	MsgRef  string              `xml:",omitempty"`
	CmdRef  string              `xml:",omitempty"`
	Cmd     string              `xml:",omitempty"`
	Target  string              `xml:"Target>LocURI"`
	Source  string              `xml:"Source>LocURI"`
	Data    string              `xml:",omitempty"`
	Item    []SyncMLCommandItem `xml:",any"`
}

type SyncMLBody struct {
	Item []SyncMLCommand `xml:",any"`
}

type SyncMLMessage struct {
	XMLinfo string       `xml:"xmlns,attr"`
	Header  SyncMLHeader `xml:"SyncHdr"`
	Body    SyncMLBody   `xml:"SyncBody"`
}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("enrollment_status"),
		table.TextColumn("enrolled_user"),
		table.TextColumn("mdm_command_input"),
		table.TextColumn("mdm_command_output"),
		table.TextColumn("raw_mdm_command_output"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// grabbing command input query if present
	var inputCmd string

	// checking if 'mdm_command_input' is in the where clause
	if constraintList, present := queryContext.Constraints["mdm_command_input"]; present {
		for _, constraint := range constraintList.Constraints {
			if constraint.Operator == table.OperatorEquals {
				inputCmd = constraint.Expression // this input as to be kept as-is and returned on the same input column due to a sqlite requirement
				log.Debug().Msgf("mdm_bridge input command request:\n%s", inputCmd)
			}
		}
	}

	// getting MDM enrollment status
	isHostMDMenrolled, enrollmentURI, err := getEnrollmentInfo()
	if err != nil {
		return nil, fmt.Errorf("there was a problem getting enrollment info: %s ", err)
	}

	// updating device enrollment status
	deviceEnrollmentStatus := "device_enrolled"
	if isHostMDMenrolled == 0 {
		deviceEnrollmentStatus = "device_not_enrolled"
	}

	// Executing the input MDM command if it was present
	if len(inputCmd) > 0 {

		// performs the actual MDM cmd execution against the OS MDM stack
		outputCmd, err := executeMDMcommand(strings.TrimSpace(inputCmd))
		if err != nil {
			return nil, fmt.Errorf("mdm command execution: %s ", err)
		}

		log.Debug().Msgf("mdm_bridge output command response:\n%s", outputCmd)

		// grabbing the command parsed command output
		minimalOutputCmd, err := getCmdResponseData(strings.TrimSpace(outputCmd))
		if err != nil {
			return nil, fmt.Errorf("mdm command response parsing: %s ", err)
		}

		return []map[string]string{
			{
				"enrollment_status":      deviceEnrollmentStatus,
				"enrolled_user":          enrollmentURI,
				"mdm_command_input":      inputCmd,
				"mdm_command_output":     minimalOutputCmd,
				"raw_mdm_command_output": outputCmd,
			},
		}, nil
	}

	// returning table enrollment status + cmd output status if present
	return []map[string]string{
		{
			"enrollment_status":      deviceEnrollmentStatus,
			"enrolled_user":          enrollmentURI,
			"mdm_command_input":      "",
			"mdm_command_ouput":      "",
			"raw_mdm_command_output": "",
		},
	}, nil
}

// dummy charset reader just to satisfy the xml decoder requirements
func identReader(encoding string, input io.Reader) (io.Reader, error) {
	return input, nil
}

// getCommandResponseData returns the response data for a given command
func getCmdResponseData(outputCmd string) (string, error) {
	var responseData string

	// creating a new SyncML message object
	messageObject := new(SyncMLMessage)

	// parsing output SyncML message
	d := xml.NewDecoder(bytes.NewReader([]byte(outputCmd)))
	d.CharsetReader = identReader

	// decoding the XML message
	if err := d.Decode(messageObject); err != nil {
		return "", err
	}

	// getting response data from output message
	if len(messageObject.Body.Item) > 0 {
		for _, element := range messageObject.Body.Item {

			// getting the results tag for the input commands
			if element.XMLName.Local != "Results" {
				continue
			}

			// results will be appended in a comma-separated list
			if len(element.Item) > 0 {

				// extracting the data from the result
				workStr := element.Item[0].Data
				if len(workStr) == 0 {
					workStr = "" // default value for empty data
				}
				responseData += workStr
			}
		}
	}

	return responseData, nil
}

// builtin windows.UTF16ToString string expects a uint16 array but we have a uint16 pointer
// so we are allocating dynamic memory and moving data around before calling windows.UTF16ToString
func localUTF16toString(ptr unsafe.Pointer) (string, error) {
	if ptr == nil {
		return "", errors.New("failed UTF16 conversion due to null pointer")
	}

	// grabbing input string length
	lenPtr, _, err := proclstrlenW.Call(uintptr(ptr))
	if err != windows.ERROR_SUCCESS {
		return "", err
	}

	// returning empty string if length is 0
	strBytesLen := int32(lenPtr) * 2 // Windows UNICODE uses 2 bytes per character
	if strBytesLen == 0 {
		return "", nil
	}

	// allocating an uint16 array buffer
	buf := make([]uint16, strBytesLen)

	// moving the data around
	_, _, err = procRtlMoveMemory.Call((uintptr)(unsafe.Pointer(&buf[0])), uintptr(ptr), uintptr(strBytesLen))
	if err != windows.ERROR_SUCCESS {
		return "", err
	}

	// and finally converting the uint16 array to a string
	return windows.UTF16ToString(buf), nil
}

// getEnrollmentInfo returns the MDM enrollment status by calling into OS API IsDeviceRegisteredWithManagement()
func getEnrollmentInfo() (uint32, string, error) {
	// variable to hold the MDM enrollment status
	var isDeviceRegisteredWithMDM uint32

	// heap-allocated buffer to hold the URI data
	buffUriData := make([]uint16, 0, maxBufSize)

	// IsDeviceRegisteredWithManagement is going to return the MDM enrollment status
	// https://learn.microsoft.com/en-us/windows/win32/api/mdmregistration/nf-mdmregistration-isdeviceregisteredwithmanagement
	if returnCode, _, err := procIsDeviceRegisteredWithManagement.Call(uintptr(unsafe.Pointer(&isDeviceRegisteredWithMDM)), maxBufSize, uintptr(unsafe.Pointer(&buffUriData))); returnCode != uintptr(windows.ERROR_SUCCESS) {
		return 0, "", fmt.Errorf("there was an error calling IsDeviceRegisteredWithManagement(): %s (0x%X)", err, returnCode)
	}

	// Sanity check to ensure that we are returning a valid string
	uriData := ""
	if isDeviceRegisteredWithMDM > 0 {
		workUriData, err := localUTF16toString(unsafe.Pointer(&buffUriData))
		if err != nil {
			return 0, "", err
		}

		if len(workUriData) > 0 {
			uriData = workUriData
		}
	}

	return isDeviceRegisteredWithMDM, uriData, nil
}

// isReadOnlyCommandRequest returns true if the verbs used on input SyncML commads are only Get
func isReadOnlyCommandRequest(inputCmd string) (bool, error) {
	if len(inputCmd) == 0 {
		return false, errors.New("empty input command")
	}

	// creating a new SyncMLBody message object
	messageObject := new(SyncMLBody)

	// parsing output SyncML message
	d := xml.NewDecoder(bytes.NewReader([]byte(inputCmd)))
	d.CharsetReader = identReader

	// decoding the XML message
	if err := d.Decode(messageObject); err != nil {
		return false, err
	}

	// sanity check on the input command structure
	if len(messageObject.Item) == 0 {
		return false, nil
	}

	// checking if input SyncML commands are only Get
	for _, element := range messageObject.Item {

		// checking if input SyncML verb is different that Get
		commandVerb := strings.ToLower(element.XMLName.Local)
		if commandVerb != "get" {
			return false, fmt.Errorf("%s is a not supported SyncML command verb", commandVerb)
		}
	}

	return true, nil
}

// Borrowed from https://stackoverflow.com/questions/53476012/how-to-validate-a-xml
func IsValidXML(s string) bool {
	return xml.Unmarshal([]byte(s), new(interface{})) == nil
}

// isValidMDMcommand checks if input SyncML command is valid
func isValidMDMcommand(inputCMD string) (bool, error) {
	// checking if input MDM command is empty
	if len(inputCMD) == 0 {
		return false, errors.New("input MDM command is empty")
	}

	// checking if input MDM command is a valid SyncML command
	isSyncBodyPrefixPresent := strings.HasPrefix(strings.ToLower(inputCMD), "<syncbody>")
	isSyncBodySuffixPresent := strings.HasSuffix(strings.ToLower(inputCMD), "</syncbody>")
	if !isSyncBodyPrefixPresent || !isSyncBodySuffixPresent {
		return false, errors.New("input MDM command is not a valid command")
	}

	// checking if input MDM command is a valid XML
	if !IsValidXML(inputCMD) {
		return false, errors.New("input MDM command is not a valid XML")
	}

	// checking if input MDM command is a read-only command
	if validCmd, err := isReadOnlyCommandRequest(inputCMD); !validCmd {
		return false, err
	}

	return true, nil
}

// executeMDMcommand executes syncML MDM commands against the OS MDM stack and returns the status of the command execution
func executeMDMcommand(inputCMD string) (output string, err error) {
	// Synchronizing MDM command execution
	mu.Lock()
	defer mu.Unlock()

	comshim.Add(1)
	defer comshim.Done()

	// checking if input MDM command is valid
	if validCommand, err := isValidMDMcommand(inputCMD); !validCommand {
		return "", err
	}

	// Enabling MDM command executions
	if err := enableCmdExecution(); err != nil {
		return "", err
	}
	defer func() {
		if disableErr := disableCmdExecution(); disableErr != nil && err == nil {
			output, err = "", disableErr
		}
	}()

	// Close MDM management mutex if neeeded - this is a hack to enable multiple MDM management calls
	handle, err := windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, windows.StringToUTF16Ptr(mdmMutexName))
	if err == nil {
		// closing handle just opened due to OpenMutex()
		if err := windows.CloseHandle(handle); err != nil {
			log.Warn().Msgf("error from CloseHandle() on handle returned from OpenMutex() for MDM mutex: (%s)", err)
		}
		// then closing previously used handle
		if err := closeManagementMutex(); err != nil {
			return "", err
		}
	}

	// converting input MDM cmd go string into UTF16 windows string
	inputCmdPtr, err := windows.UTF16FromString(inputCMD)
	if err != nil {
		return "", err
	}

	// MDM stack is ready to receive commands
	// The code below is just using returnCode to determine if call was successul or not. The err
	// variable returns status above call dispatching so it not needed and actually introduce
	// confusion about the status of the call.
	var outputStrBuffer *uint16
	if returnCode, _, _ := procSendSyncMLcommand.Call(uintptr(unsafe.Pointer(&inputCmdPtr[0])), uintptr(unsafe.Pointer(&outputStrBuffer))); returnCode != uintptr(windows.ERROR_SUCCESS) {
		return "", fmt.Errorf("there was an error calling ApplyLocalManagementSyncML(): (0x%X)", returnCode)
	}

	// converting Windows MDM UTF16 output string into go string
	outputCmd, err := localUTF16toString(unsafe.Pointer(outputStrBuffer))
	if err != nil {
		return "", err
	}

	// freeing OS allocated heap memory
	_, err = windows.LocalFree(windows.Handle(unsafe.Pointer(outputStrBuffer)))
	if err != nil {
		return "", err
	}

	if len(outputCmd) == 0 {
		return "", errors.New("the OS MDM stack returned an empty string")
	}

	return outputCmd, nil
}

// closeManagementMutex walks the system handles to find and close the MDM management mutexes on
// current process. This is a hack found after reverse engineering mdmlocalmanagement.dll.
func closeManagementMutex() error {
	const bufsize = 2048 // buffer allocation for native windows syscalls
	// Disabling uint32->int conversion error from gosec below because under the covers os.Getpid() is converting a uint32
	// returned by kernel32 to an int, so this is safe
	currentProcessPID := uint32(os.Getpid()) // nolint:gosec

	var handleOccurences uint32

	// querying first the list of handles on the kernel using NtQuerySystemInformation() syscall and SystemHandleInformation
	// https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntquerysysteminformation
	bufQuerySystemSyscall := make([]byte, 0, bufsize)
	var rlen uint32
	if st := ntdll.CallWithExpandingBuffer(func() ntdll.NtStatus {
		return ntdll.NtQuerySystemInformation(
			ntdll.SystemHandleInformation,
			&bufQuerySystemSyscall[0],
			// This will never return enough data to cause an overflow so int->uint32 conversion warning disabled
			uint32(len(bufQuerySystemSyscall)), // nolint:gosec
			&rlen,
		)
	}, &bufQuerySystemSyscall, &rlen); st.IsError() {
		return fmt.Errorf("NtQuerySystemInformation: %s, len=%d", st.Error(), rlen)
	}

	// Sanity check on returned buffer
	if bufQuerySystemSyscall == nil {
		return errors.New("invalid list of handles returned by NtQuerySystemInformation")
	}

	// Casting the returned buffer to the SystemHandleInformation type
	// https://www.geoffchappell.com/studies/windows/km/ntoskrnl/api/ex/sysinfo/handle.htm
	handlesList := (*ntdll.SystemHandleInformationT)(unsafe.Pointer(&bufQuerySystemSyscall[0]))

	// Iterating over the list of handlers
	for _, systemHandleEntry := range handlesList.GetHandles() {

		// only processing the current process handles
		if currentProcessPID != systemHandleEntry.OwnerPid {
			continue
		}

		// Calling NtQueryObject syscalls with ObjectTypeInformation to obtain object type information of a given handle. This requires static allocation.
		// https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntqueryobject
		var handleObjectTypeBuf [bufsize]byte
		var outputLen uint32
		st := ntdll.NtQueryObject(ntdll.Handle(systemHandleEntry.HandleValue), ntdll.ObjectTypeInformation, &handleObjectTypeBuf[0], uint32(len(handleObjectTypeBuf)), &outputLen)
		if st != ntdll.STATUS_SUCCESS || outputLen == 0 {
			continue
		}

		// Casting the returned buffer to the OBJECT_TYPE_INFORMATION type
		// https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/ns-ntifs-__public_object_type_information
		oti := (*ntdll.ObjectTypeInformationT)(unsafe.Pointer(&handleObjectTypeBuf[0]))
		if oti.TypeName.String() == "Mutant" {

			// Calling NtQueryObject syscalls with ObjectNameInformation to obtain named object information of a given handle. This requires static allocation.
			// https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntqueryobject
			var handleObjectNameBuf [bufsize]byte
			var outputLen uint32
			st := ntdll.NtQueryObject(ntdll.Handle(systemHandleEntry.HandleValue), ntdll.ObjectNameInformation, &handleObjectNameBuf[0], uint32(len(handleObjectNameBuf)), &outputLen)
			if st != ntdll.STATUS_SUCCESS || outputLen == 0 {
				continue
			}

			oni := (*ntdll.ObjectNameInformationT)(unsafe.Pointer(&handleObjectNameBuf[0]))

			if strings.Contains(oni.Name.String(), mdmMutexName) {
				_ = windows.CloseHandle(windows.Handle(systemHandleEntry.HandleValue))
				handleOccurences++
			}
		}
	}

	if handleOccurences == 0 {
		return fmt.Errorf("target named mutex %s was not found", mdmMutexName)
	}

	return nil
}

// embeddedModeFallback is what Windows hashes for the Flags check when its SMBIOS walk misses the type 1 structure.
var embeddedModeFallback = []byte{0x57, 0x26, 0xdb, 0xcb, 0x8d, 0xce, 0x27, 0x47, 0x9d, 0x45, 0x1e, 0x18, 0x66, 0xf3, 0x7e, 0x21}

// findSMBIOSUUID returns the raw UUID bytes of the first type 1 structure in an SMBIOS structure table, or nil. With windowsQuirk
// it mirrors the walk in embeddedmodesvcapi.dll, which never resets its "previous byte was zero" state between structures: a
// string-less structure after the first one ends one byte early, and the walk can drift past type 1 or onto a stray 0x01 byte.
func findSMBIOSUUID(smbiosTable []byte, windowsQuirk bool) []byte {
	prevZero := false
	for i := 0; i+1 < len(smbiosTable); {
		if smbiosTable[i] == 1 {
			if i+24 > len(smbiosTable) {
				return nil
			}
			return smbiosTable[i+8 : i+24]
		}
		if !windowsQuirk {
			prevZero = false
		}
		// Skip the formatted area, then find the double zero that ends the string set.
		j := i + int(smbiosTable[i+1])
		for ; j < len(smbiosTable) && (smbiosTable[j] != 0 || !prevZero); j++ {
			prevZero = smbiosTable[j] == 0
		}
		if j >= len(smbiosTable) {
			return nil
		}
		i = j + 1
	}
	return nil
}

// getFlagsValue returns the EmbeddedMode Flags value that Windows accepts: SHA-256 of the type 1 UUID as found by Windows' own
// SMBIOS walk. If that differs from the spec-compliant UUID, IsEmbeddedModeAllowed picks between them, so a Windows build that
// fixes its walk keeps working.
func getFlagsValue() ([]byte, error) {
	stream, _, err := smbios.Stream()
	if err != nil {
		return nil, fmt.Errorf("opening SMBIOS table: %w", err)
	}
	defer stream.Close()
	smbiosTable, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("reading SMBIOS table: %w", err)
	}

	windowsUUID := findSMBIOSUUID(smbiosTable, true)
	if windowsUUID == nil {
		windowsUUID = embeddedModeFallback
	}
	windowsHash := sha256.Sum256(windowsUUID)
	specUUID := findSMBIOSUUID(smbiosTable, false)
	if specUUID == nil || bytes.Equal(specUUID, windowsUUID) {
		return windowsHash[:], nil
	}
	specHash := sha256.Sum256(specUUID)

	// The walks disagree, so ask Windows. It re-reads Flags on every IsEmbeddedModeAllowed call. Probe failures fall back to
	// Windows' walk rather than failing, since this runs once per process.
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, flagsRegistryLocation, registry.SET_VALUE)
	if err == nil && procIsEmbeddedModeAllowed.Find() == nil {
		defer k.Close()
		// Remove the probe value; enableCmdExecution writes the chosen one before each command.
		defer func() { _ = k.DeleteValue(flagsRegistryName) }()
		for i, candidate := range [][]byte{windowsHash[:], specHash[:]} {
			if err := k.SetBinaryValue(flagsRegistryName, candidate); err != nil {
				break
			}
			var allowed int32
			if hr, _, _ := procIsEmbeddedModeAllowed.Call(uintptr(unsafe.Pointer(&allowed))); hr != 0 {
				break
			}
			if allowed != 0 {
				log.Info().Bool("windows_walk", i == 0).Msg("mdm_bridge: SMBIOS walks disagree, Windows confirmed the EmbeddedMode Flags value")
				return candidate, nil
			}
		}
	}
	log.Warn().Msg("mdm_bridge: SMBIOS walks disagree and Windows confirmed neither EmbeddedMode Flags value, using Windows' walk")
	return windowsHash[:], nil
}

// enableCmdExecution initializes the registry flags required for OS MDM execution
func enableCmdExecution() error {
	flagsValueInit.Do(func() {
		value, err := getFlagsValue()
		if err != nil {
			log.Error().Err(err).Msg("there was an issue generating the EmbeddedMode Flags value")
			return
		}
		flagsValue = value
	})

	if len(flagsValue) == 0 {
		return errors.New("there was a problem generating the EmbeddedMode Flags value")
	}

	// Flags value is already there, so we just need to write the registry to enable MDM commands
	// execution. Registry flag is set and unset on each command execution to isolate the MDM command
	// execution to this logic only
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, flagsRegistryLocation, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}

	if err := k.SetBinaryValue(flagsRegistryName, flagsValue); err != nil {
		return err
	}

	if err := k.Close(); err != nil {
		return err
	}

	// Registering is gated by Flags too, so it must follow the write above. ApplyLocalManagementSyncML fails with
	// ERROR_INVALID_STATE until a registration succeeds, which Windows persists.
	mdmManagementStackInit.Do(func() {
		if returnCode, _, _ := procRegisterDeviceWithLocalManagement.Call(uintptr(unsafe.Pointer(nil))); returnCode != uintptr(windows.ERROR_SUCCESS) {
			log.Error().Msgf("there was an error calling RegisterDeviceWithLocalManagement(): (0x%X)", returnCode)
		}
	})

	return nil
}

// enableCmdExecution removes a special registry flag to disable MDM command execution
func disableCmdExecution() error {
	// Here we are just making sure to delete the Flags registry entry
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, flagsRegistryLocation, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}

	if err := k.DeleteValue(flagsRegistryName); err != nil {
		return err
	}

	if err := k.Close(); err != nil {
		return err
	}

	return nil
}
