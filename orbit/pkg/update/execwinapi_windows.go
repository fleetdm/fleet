//go:build windows

package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	dllMDMRegistration *windows.LazyDLL = windows.NewLazySystemDLL("mdmregistration.dll")

	// RegisterDeviceWithManagement registers a device with a MDM service:
	// https://learn.microsoft.com/en-us/windows/win32/api/mdmregistration/nf-mdmregistration-registerdevicewithmanagement
	procRegisterDeviceWithManagement *windows.LazyProc = dllMDMRegistration.NewProc("RegisterDeviceWithManagement")
)

// Exported so that it can be used in tools/ (so that it can be built for
// Windows and tested on a Windows machine). Otherwise not meant to be called
// from outside this package.
func RunWindowsMDMEnrollment(args WindowsMDMEnrollmentArgs) error {
	installType, err := readInstallationType()
	if err != nil {
		return err
	}
	if strings.ToLower(installType) == "server" {
		// do not enroll, it is a server
		return errIsWindowsServer
	}
	return enrollHostToMDM(args)
}

func readInstallationType() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("InstallationType")
	if err != nil {
		return "", err
	}
	return s, nil
}

// TODO(mna): refactor to a Windows-specific package to constrain usage of
// unsafe to that package, once https://github.com/fleetdm/fleet/pull/12387
// lands.

// Perform the host MDM enrollment process using MS-MDE protocol:
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/5c841535-042e-489e-913c-9d783d741267
func enrollHostToMDM(args WindowsMDMEnrollmentArgs) error {
	discoveryURLPtr, err := syscall.UTF16PtrFromString(args.DiscoveryURL)
	if err != nil {
		return fmt.Errorf("discovery URL to UTF16 pointer: %w", err)
	}

	// we use an empty user UPN, it is not a required argument
	userPtr, err := syscall.UTF16PtrFromString("")
	if err != nil {
		return fmt.Errorf("user UPN to UTF16 pointer: %w", err)
	}

	accessTok, err := generateWindowsMDMAccessTokenPayload(args)
	if err != nil {
		return fmt.Errorf("generate access token payload: %w", err)
	}
	accessTokPtr, err := syscall.UTF16PtrFromString(string(accessTok))
	if err != nil {
		return fmt.Errorf("access token to UTF16 pointer: %w", err)
	}

	// pre-load the DLL and pre-find the procedure, to return a more meaningful
	// message if those steps fail and avoid a panic.
	if err := dllMDMRegistration.Load(); err != nil {
		return fmt.Errorf("load MDM dll: %w", err)
	}
	if err := procRegisterDeviceWithManagement.Find(); err != nil {
		return fmt.Errorf("find MDM RegisterDeviceWithManagement procedure: %w", err)
	}

	if code, _, err := procRegisterDeviceWithManagement.Call(
		uintptr(unsafe.Pointer(userPtr)),
		uintptr(unsafe.Pointer(discoveryURLPtr)),
		uintptr(unsafe.Pointer(accessTokPtr)),
	); code != uintptr(windows.ERROR_SUCCESS) {
		// hexadecimal error code can help identify error here:
		//   https://learn.microsoft.com/en-us/windows/win32/mdmreg/mdm-registration-constants
		// decimal error code can help identify error here (look for the ERROR_xxx constants):
		//   https://pkg.go.dev/golang.org/x/sys/windows#pkg-constants
		//
		// Note that the error message may be "The operation completed
		// successfully." even though there is an error (e.g. if the discovery URL
		// results in a 404 not found, the error code will be 0x80190194 which
		// means windows.HTTP_E_STATUS_NOT_FOUND). In this case, translate the
		// message to something more useful.
		if httpCode := code - uintptr(windows.HTTP_E_STATUS_BAD_REQUEST); httpCode >= 0 && httpCode < 200 {
			// status bad request is 400, so if error code is between 400 and < 600.
			err = fmt.Errorf("using discovery URL %q: HTTP error code %d", args.DiscoveryURL, http.StatusBadRequest+httpCode)
		}
		return fmt.Errorf("RegisterDeviceWithManagement failed: %s (%#x - %[2]d)", err, code)
	}

	return nil
}

// TODO(mna): move this to fleet/server/fleet package once
// https://github.com/fleetdm/fleet/pull/12387 merged as it will be needed by
// the server too.

type accessTokenPayload struct {
	// Type is the enrollment type, such as "programmatic".
	Type    windowsMDMEnrollmentType `json:"type"`
	Payload struct {
		HostUUID string `json:"host_uuid"`
	} `json:"payload"`
}

type windowsMDMEnrollmentType int

const (
	windowsMDMProgrammaticEnrollmentType windowsMDMEnrollmentType = 1
)

func generateWindowsMDMAccessTokenPayload(args WindowsMDMEnrollmentArgs) ([]byte, error) {
	var pld accessTokenPayload

	pld.Type = windowsMDMProgrammaticEnrollmentType // always programmatic for now
	pld.Payload.HostUUID = args.HostUUID
	return json.Marshal(pld)
}
