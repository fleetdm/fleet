//go:build !darwin
// +build !darwin

package user

// CheckUserGuiLoginStatus returns whether or not a user is logged into the machine via the GUI.
func IsUserLoggedInViaGui() (bool, error) {
	return true, nil
}
