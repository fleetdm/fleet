package externalrefs

import (
	"fmt"
	"strings"

	macoffice "github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
)

// MicrosoftVersionFromReleaseNotes returns the short version from the release notes given a Homebrew version.
// Homebrew version "16.95.25032931"
// Release notes version "Version 16.95.3 (Build 25032931)"
// For this example it would return "16.95.3"
func MicrosoftVersionFromReleaseNotes(args ...interface{}) (string, error) {
	if len(args) > 0 {
		if homebrewVersion, ok := args[0].(string); ok {
			versionParts := strings.Split(homebrewVersion, ".")              // homebrew version format is like "16.95.25032931"
			version := strings.Join(versionParts[:len(versionParts)-1], ".") // Extract version without the build number
			build := versionParts[len(versionParts)-1]                       // Extract the build number

			releaseNotes, err := macoffice.GetReleaseNotes(true)
			if err != nil {
				return "", fmt.Errorf("failed to retrieve release notes: %w", err)
			}

			for _, relNote := range releaseNotes {
				shortVersion := relNote.ShortVersionFormat()
				if strings.HasPrefix(shortVersion, version) && relNote.BuildNumber() == build {
					return shortVersion, nil
				}
			}
		}
	}
	return "", fmt.Errorf("input version not provided")
}
