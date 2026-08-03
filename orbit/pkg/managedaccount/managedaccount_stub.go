//go:build !windows

package managedaccount

import "errors"

// provisionAccount is a placeholder for non-Windows builds.
func provisionAccount(username, password string) error {
	return errors.New("managed local account provisioning is only supported on Windows")
}
