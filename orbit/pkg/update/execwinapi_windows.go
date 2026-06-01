//go:build windows

// nolint:gosec,G103 // Reason: unsafe required for Windows API calls.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	if httpCode := code - uintptr(windows.HTTP_E_STATUS_BAD_REQUEST); httpCode < 200 {
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

// fleetMDMProviderID must match syncml.DocProvisioningAppProviderID, the provider ID Fleet sets in the Windows
// MDM enrollment provisioning doc. It is the value of the ProviderID registry value on Fleet's enrollment.
const fleetMDMProviderID = "Fleet"

// TriggerWindowsMDMSync starts an on-demand, client-initiated OMA-DM session with the Fleet MDM server so that
// queued Windows MDM commands are delivered without waiting for the device's next scheduled poll. It runs the OS
// deviceenroller for Fleet's enrollment in client-initiated mode: `deviceenroller.exe /o <EnrollmentGUID> /c`.
// This is deliberately the client-initiated path, not the push-initiated path (`/c /z`), which fails with Event
// 4603 (GetPushAlertInfo / notificationIdNotRetrieved) on current Windows builds.
//
// Exported so it can be built/tested for Windows from tools/. Not meant to be called from outside this package.
func TriggerWindowsMDMSync() error {
	guid, err := fleetMDMEnrollmentGUID()
	if err != nil {
		return fmt.Errorf("find Fleet MDM enrollment: %w", err)
	}

	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	// orbit is a 64-bit process, so System32 resolves to the real (64-bit) System32 that contains
	// deviceenroller.exe; there is no WOW64 redirection to account for.
	deviceEnroller := filepath.Join(systemRoot, "System32", "deviceenroller.exe")

	cmd := exec.Command(deviceEnroller, "/o", guid, "/c")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("run deviceenroller /o %s /c: %w (output: %q)", guid, err, string(out))
	}
	return nil
}

// fleetMDMEnrollmentGUID returns the enrollment GUID of the active Fleet Windows MDM enrollment by scanning
// HKLM\SOFTWARE\Microsoft\Enrollments for the subkey whose ProviderID is Fleet's and whose EnrollmentState is
// active. The subkey name is the enrollment GUID that deviceenroller's /o argument expects.
func fleetMDMEnrollmentGUID() (string, error) {
	const enrollmentsPath = `SOFTWARE\Microsoft\Enrollments`
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, enrollmentsPath, registry.READ)
	if err != nil {
		return "", fmt.Errorf("open enrollments registry key: %w", err)
	}
	defer root.Close()

	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return "", fmt.Errorf("read enrollment subkeys: %w", err)
	}

	const enrollmentStateActive = 1
	for _, name := range names {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, enrollmentsPath+`\`+name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		providerID, _, providerErr := k.GetStringValue("ProviderID")
		state, _, stateErr := k.GetIntegerValue("EnrollmentState")
		k.Close()
		if providerErr == nil && stateErr == nil && providerID == fleetMDMProviderID && state == enrollmentStateActive {
			return name, nil
		}
	}
	return "", errors.New("no active Fleet MDM enrollment found in registry")
}
