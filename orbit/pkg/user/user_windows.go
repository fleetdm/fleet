//go:build windows
// +build windows

package user

// UserLoggedInViaGui returns the name of the user logged into the machine via the GUI. This
// function is only relevant on MacOS, where it is used to prevent errors when launching Fleet
// Desktop. For other platforms we return an empty string which can be ignored.
func UserLoggedInViaGui() (*string, error) {
	user := ""
	return &user, nil
}
