//go:build windows

package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"unsafe"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	dllMDMRegistration *windows.LazyDLL = windows.NewLazySystemDLL("mdmregistration.dll")

	// RegisterDeviceWithManagement registers a device with a MDM service:
	// https://learn.microsoft.com/en-us/windows/win32/api/mdmregistration/nf-mdmregistration-registerdevicewithmanagement
	procRegisterDeviceWithManagement *windows.LazyProc = dllMDMRegistration.NewProc("RegisterDeviceWithManagement")

	// UnregisterDeviceWithManagement unregisters a device from a MDM service:
	// https://learn.microsoft.com/en-us/windows/win32/api/mdmregistration/nf-mdmregistration-unregisterdevicewithmanagement
	procUnregisterDeviceWithManagement *windows.LazyProc = dllMDMRegistration.NewProc("UnregisterDeviceWithManagement")
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

// Exported so that it can be used in tools/ (so that it can be built for
// Windows and tested on a Windows machine). Otherwise not meant to be called
// from outside this package.
func RunWindowsMDMUnenrollment(args WindowsMDMEnrollmentArgs) error {
	installType, err := readInstallationType()
	if err != nil {
		return err
	}
	if strings.ToLower(installType) == "server" {
		// do not unenroll, it is a server
		return errIsWindowsServer
	}
	return unenrollHostFromMDM()
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
	// message if those steps fail and avoid a panic (those are no-ops once
	// loaded/found).
	if err := dllMDMRegistration.Load(); err != nil {
		return fmt.Errorf("load MDM dll: %w", err)
	}
	if err := procRegisterDeviceWithManagement.Find(); err != nil {
		return fmt.Errorf("find MDM RegisterDeviceWithManagement procedure: %w", err)
	}

	code, _, err := procRegisterDeviceWithManagement.Call(
		uintptr(unsafe.Pointer(userPtr)),
		uintptr(unsafe.Pointer(discoveryURLPtr)),
		uintptr(unsafe.Pointer(accessTokPtr)),
	)
	log.Debug().Msgf("RegisterDeviceWithManagement returned code: %#x ; message: %v", code, err)
	if code != uintptr(windows.ERROR_SUCCESS) {
		return improveWindowsAPIError("RegisterDeviceWithManagement", args.DiscoveryURL, code, err)
	}

	return nil
}

// Perform the host MDM unenrollment process using MS-MDE protocol:
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde/5c841535-042e-489e-913c-9d783d741267
func unenrollHostFromMDM() error {
	// pre-load the DLL and pre-find the procedure, to return a more meaningful
	// message if those steps fail and avoid a panic (those are no-ops once
	// loaded/found).
	if err := dllMDMRegistration.Load(); err != nil {
		return fmt.Errorf("load MDM dll: %w", err)
	}
	if err := procUnregisterDeviceWithManagement.Find(); err != nil {
		return fmt.Errorf("find MDM UnregisterDeviceWithManagement procedure: %w", err)
	}

	// must explicitly pass 0 here, see for details:
	// https://github.com/fleetdm/fleet/issues/12342#issuecomment-1608190367
	code, _, err := procUnregisterDeviceWithManagement.Call(0)
	log.Debug().Msgf("UnregisterDeviceWithManagement returned code: %#x ; message: %v", code, err)
	if code != uintptr(windows.ERROR_SUCCESS) {
		return improveWindowsAPIError("UnregisterDeviceWithManagement", "", code, err)
	}

	return nil
}

func improveWindowsAPIError(apiFunc, discoURL string, code uintptr, err error) error {
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
		if discoURL != "" {
			err = fmt.Errorf("using discovery URL %q: HTTP error code %d", discoURL, http.StatusBadRequest+httpCode)
		} else {
			err = fmt.Errorf("HTTP error code %d", http.StatusBadRequest+httpCode)
		}
	}
	return fmt.Errorf("%s failed: %s (%#x - %[3]d)", apiFunc, err, code)
}

func generateWindowsMDMAccessTokenPayload(args WindowsMDMEnrollmentArgs) ([]byte, error) {
	var pld fleet.WindowsMDMAccessTokenPayload
	pld.Type = fleet.WindowsMDMProgrammaticEnrollmentType // always programmatic for now
	pld.Payload.OrbitNodeKey = args.OrbitNodeKey
	return json.Marshal(pld)
}

// IsRunningOnWindowsServer determines if the process is running on a Windows
// server. Exported so it can be used across packages.
func IsRunningOnWindowsServer() (bool, error) {
	installType, err := readInstallationType()
	if err != nil {
		return false, err
	}

	if strings.ToLower(installType) == "server" {
		return true, nil
	}

	return false, nil
}
