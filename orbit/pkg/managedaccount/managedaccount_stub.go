//go:build !windows

package managedaccount

import "errors"

// provisionAccount is a placeholder for non-Windows builds. The receiver is only registered on
// Windows, and the server only sends the notification to Windows hosts, so reaching this is a bug
// rather than a supported path.
func provisionAccount(username, password string) error {
	return errors.New("managed local account provisioning is only supported on Windows")
}
