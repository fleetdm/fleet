//go:build windows
// +build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modmdmregistration *windows.LazyDLL = windows.NewLazySystemDLL("mdmregistration.dll")

	procIsDeviceRegisteredWithManagement *windows.LazyProc = modmdmregistration.NewProc("IsDeviceRegisteredWithManagement")
	procRegisterDeviceWithManagement     *windows.LazyProc = modmdmregistration.NewProc("RegisterDeviceWithManagement")
)

const (
	maxBufSize = 2048 // Max Unicode Length for UPN - https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-adls/63f5e067-d1b3-4e6e-9e53-a92953b6005b
)

// checking for administrator rights
func checkForAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}

	f.Close()

	return true
}

// builtin utf16tostring string expects a uint16 array but we have a pointer to a uint16
// so we need to cast it after converting it to an unsafe pointer
// this is a common pattern though the buffer size varies
// see https://golang.org/pkg/unsafe/#Pointer for more details
func localUTF16toString(ptr unsafe.Pointer) (string, error) {
	if ptr == nil {
		return "", errors.New("failed UTF16 conversion due to null pointer")
	}
	uint16ptrarr := (*[maxBufSize]uint16)(ptr)[:]
	return windows.UTF16ToString(uint16ptrarr), nil
}

// getting the MDM enrollment status by calling into OS API IsDeviceRegisteredWithManagement()
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

	uriData, err := localUTF16toString(unsafe.Pointer(&buffUriData))
	if err != nil {
		return 0, "", err
	}

	return isDeviceRegisteredWithMDM, uriData, nil
}

// Perform the host MDM enrollment process using MS-MDE protocol
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/5c841535-042e-489e-913c-9d783d741267
func enrollHostToMDM(mdmDiscoveryEndpoint string, mdmUpnUser string) (bool, error) {
	// converting go server discovery endpoint string into UTF16 windows string
	inputDiscoveryEndpoint := syscall.StringToUTF16Ptr(mdmDiscoveryEndpoint)

	// converting go upn user string into UTF16 windows string
	inputMdmUPN := syscall.StringToUTF16Ptr(mdmUpnUser)

	// converting go csr string into UTF16 windows string
	// passing empty value to force MDM OS stack to generate a CSR for us
	// See here for details of CSR generation https://github.com/Gerenios/AADInternals/blob/2efc23e28dc66135d37eca117f456c64b2e48ea9/MDM_utils.ps1#L84
	// CN should be <randomGUID>|<DeviceClientId>
	// DeviceClientId value is located at HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Provisioning\OMADM\MDMDeviceID
	var encodedB64 string
	inputCSRreq := syscall.StringToUTF16Ptr(encodedB64)

	// RegisterDeviceWithManagement() registers a device with a MDM service

	// https://learn.microsoft.com/en-us/windows/win32/api/mdmregistration/nf-mdmregistration-registerdevicewithmanagement
	if returnCode, _, err := procRegisterDeviceWithManagement.Call(uintptr(unsafe.Pointer(inputMdmUPN)), uintptr(unsafe.Pointer(inputDiscoveryEndpoint)), uintptr(unsafe.Pointer(inputCSRreq))); returnCode != uintptr(windows.ERROR_SUCCESS) {
		return false, fmt.Errorf("there was an error calling RegisterDeviceWithManagement(): %s (0x%X)", err, returnCode)
	}

	return true, nil
}

func main() {
	targetDiscoveryUrl := flag.String("discovery-url", "", "Target MDM discovery webservice url")
	targetUserEmail := flag.String("upn-email", "", "Target email to enroll server")

	flag.Usage = func() {
		w := flag.CommandLine.Output()

		fmt.Fprintf(w, "Available Command Line Options\n")
		flag.PrintDefaults()

		fmt.Fprintf(w, "\nExample Usage\n%s -discovery-url <target_mdm_server_discovery_url> -upn-email <target_mdm_enrollment_user>\n", os.Args[0])
		fmt.Fprintf(w, "%s -discovery-url https://mdmwindows.com/EnrollmentServer/Discovery.svc -upn-email demo@mdmwindows.com\n", os.Args[0])
	}

	flag.Parse()

	if len(*targetDiscoveryUrl) == 0 || len(*targetUserEmail) == 0 {
		fmt.Printf("There was a problem with provided arguments.")
		os.Exit(1)
	}

	// Checking if running with admin rights
	if !checkForAdmin() {
		fmt.Printf("You should have Administrator rights to run the tool.")
		os.Exit(1)
	}

	// Checking if host is already enrolled
	enrollmentStatus, enrollmentUser, err := getEnrollmentInfo()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	if enrollmentStatus != 0 {
		fmt.Printf("Host is already enrolled through user: %s\n", enrollmentUser)
		os.Exit(1)
	}

	// Perform the actual enrollment
	hostEnrolled, err := enrollHostToMDM(*targetDiscoveryUrl, *targetUserEmail)

	if hostEnrolled {
		fmt.Printf("Host was successfully enrolled into MDM server %s through user %s\n", *targetDiscoveryUrl, *targetUserEmail)
	} else {
		fmt.Printf(" There was a problem enrolling into MDM server %s through user %s - Error %s\n", *targetDiscoveryUrl, *targetUserEmail, err)
	}
}
