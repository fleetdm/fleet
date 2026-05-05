//go:build windows

package user

// UserLoggedInViaGui returns the name of the user logged into the machine via the GUI. This
// function is only relevant on MacOS and Linux, where it is used to prevent errors when launching Fleet Desktop.
func UserLoggedInViaGui() (*string, error) {
	user := ""
	return &user, nil
}
