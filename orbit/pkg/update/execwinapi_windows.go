//go:build windows

package update

import (
	"fmt"
	"os/user"
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
func RunWindowsMDMEnrollment(discoveryURL string) error {
	installType, err := readInstallationType()
	if err != nil {
		return err
	}
	if strings.ToLower(installType) == "server" {
		// do not enroll, it is a server
		return errIsWindowsServer
	}
	return enrollHostToMDM(discoveryURL)
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
func enrollHostToMDM(discoveryURL string) error {
	userInfo, err := user.Current() // TODO(mna): replace with the actual user we need
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	fmt.Printf("Current user: %+v\n", userInfo)

	discoveryURLPtr := syscall.StringToUTF16Ptr(discoveryURL)
	userPtr := syscall.StringToUTF16Ptr(userInfo.Username)

	fmt.Println("Try to load mdm dll in memory...")
	if err := dllMDMRegistration.Load(); err != nil {
		fmt.Println("Failed to load mdm dll in memory.")
		return fmt.Errorf("load MDM dll: %w", err)
	}
	fmt.Println("Succeeded to load mdm dll in memory!")

	fmt.Println("Try to find mdm proc in memory...")
	if err := procRegisterDeviceWithManagement.Find(); err != nil {
		fmt.Println("Failed to find mdm proc in memory.")
		return fmt.Errorf("find MDM RegisterDeviceWithManagement procedure: %w", err)
	}
	fmt.Println("Succeeded to find mdm proc in memory!")

	// converting go csr string into UTF16 windows string
	// passing empty value to force MDM OS stack to generate a CSR for us
	// See here for details of CSR generation https://github.com/Gerenios/AADInternals/blob/2efc23e28dc66135d37eca117f456c64b2e48ea9/MDM_utils.ps1#L84
	// CN should be <randomGUID>|<DeviceClientId>
	// DeviceClientId value is located at HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Provisioning\OMADM\MDMDeviceID
	var encodedB64 string = ""
	inputCSRreq := syscall.StringToUTF16Ptr(encodedB64)

	if code, _, err := procRegisterDeviceWithManagement.Call(
		uintptr(unsafe.Pointer(userPtr)),
		uintptr(unsafe.Pointer(discoveryURLPtr)),
		uintptr(unsafe.Pointer(inputCSRreq)),
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
			err = fmt.Errorf("using discovery URL %q: HTTP error code %d", discoveryURL, 400+httpCode)
		}
		return fmt.Errorf("RegisterDeviceWithManagement failed: %s (%#x - %[2]d)", err, code)
	}

	return nil
}
