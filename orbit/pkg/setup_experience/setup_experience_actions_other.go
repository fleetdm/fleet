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

func hideDesktop(_ uint32) error {
	return errors.New("hiding the desktop is only supported on Windows")
}

func showDesktop() error {
	return errors.New("showing the desktop is only supported on Windows")
}

func provisioningAccountExists() bool { return false }

func teardownProvisioningAccount(_ string, _ string) error {
	return errors.New("provisioning account teardown is only supported on Windows")
}
