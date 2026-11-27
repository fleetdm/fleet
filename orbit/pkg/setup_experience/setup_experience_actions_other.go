//go:build !windows

package setupexperience

import "errors"

func createLocalAccount(_, _, _ string) error {
	return errors.New("creating a local account is only supported on Windows")
}

func minPasswordLength() int { return 0 }

func openPINEnrollment() error {
	return errors.New("Windows Hello PIN enrollment is only supported on Windows")
}

// HideDesktop is only supported on Windows.
func HideDesktop(_ uint32) error {
	return errors.New("hiding the desktop is only supported on Windows")
}

func localAccountExists(_ string) bool { return false }

func localAccountEnabled(_ string) bool { return false }

func teardownProvisioningAccount(_ string, _ string) error {
	return errors.New("provisioning account teardown is only supported on Windows")
}

func applyProvisioningPackage(_ string) error {
	return errors.New("provisioning packages are only supported on Windows")
}

func entraJoined() bool { return false }

func restartWindows() error {
	return errors.New("restart is only supported on Windows")
}

func blankShell(_ string) error {
	return errors.New("changing the shell is only supported on Windows")
}
