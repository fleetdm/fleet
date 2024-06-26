//go:build !darwin
// +build !darwin

package user

// IsUserLoggedInViaGui returns whether or not a user is logged into the machine via the GUI. This
// function is only relevant on MacOS, where it is used to prevent errors when launching Fleet
// Desktop. We assume yes (effectively a no-op) on all other platforms.
func IsUserLoggedInViaGui() (bool, error) {
	return true, nil
}
