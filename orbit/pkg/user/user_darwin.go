//go:build darwin
// +build darwin

package user

import (
	"errors"
	"os/exec"
	"strings"
)

// IsUserLoggedInViaGui returns whether or not a user is logged into the machine via the GUI.
func IsUserLoggedInViaGui() (bool, error) {
	output, err := exec.Command("/usr/bin/stat", "-f", "%Su", "/dev/console").Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return false, errors.Join(err, errors.New(string(ee.Stderr)))
		}

		return false, err
	}

	// If no user is logged in via GUI, the command line returns "root".
	if strings.TrimSpace(string(output)) == "root" {
		return false, nil
	}

	return true, nil
}
