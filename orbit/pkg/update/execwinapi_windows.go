//go:build windows

package update

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Exported so that it can be used in tools/ (so that it can be built for
// Windows and tested on a Windows machine). Otherwise not meant to be called
// from outside this package.
func RunWindowsMDMEnrollment() error {
	installType, err := readInstallationType()
	if err != nil {
		return err
	}
	if strings.ToLower(installType) == "server" {
		// do not enroll, it is a server
		return errIsWindowsServer
	}
	return nil
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
