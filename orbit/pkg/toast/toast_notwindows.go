//go:build !windows

package toast

import "errors"

// Show is only implemented on Windows.
func Show(Notification) error {
	return errors.ErrUnsupported
}

// Remove is only implemented on Windows.
func Remove(string, string) error {
	return errors.ErrUnsupported
}

// RegisterFleetDesktopAppID is only implemented on Windows.
func RegisterFleetDesktopAppID(string, []byte) error {
	return errors.ErrUnsupported
}
