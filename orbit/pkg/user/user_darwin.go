//go:build darwin
// +build darwin

package user

import (
	"bytes"
	"os/exec"
	"regexp"
)

var re = regexp.MustCompile(`\s+Name : (\S+)`)

// UserLoggedInViaGui returns the name of the user logged into the machine via the GUI.
func UserLoggedInViaGui() (*string, error) {
	// Attempt to get the console user.
	cmd := exec.Command("/bin/sh", "-c", `scutil <<< "show State:/Users/ConsoleUser"`)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// Extract all "Name : username" entries, and return the first one that
	// isn't _mbsetupuser (if any).
	matches := re.FindAllStringSubmatch(out.String(), -1)

	for _, match := range matches {
		if len(match) > 1 && match[1] != "" && match[1] != "_mbsetupuser" {
			return &match[1], nil
		}
	}

	// No valid user found
	return nil, nil
}
