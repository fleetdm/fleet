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
		return err
	}
	discoveryURLPtr := syscall.StringToUTF16Ptr(discoveryURL)
	userPtr := syscall.StringToUTF16Ptr(userInfo.Username)

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
		return fmt.Errorf("RegisterDeviceWithManagement failed: %s (%#x)", err, code)
	}

	return nil
}
