package externalrefs

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	macoffice "github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
)

var (
	releaseNotesCache macoffice.ReleaseNotes
	cacheOnce         sync.Once
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

			var err error
			cacheOnce.Do(func() {
				releaseNotesCache, err = macoffice.GetReleaseNotes(true)
			})
			if err != nil {
				return "", fmt.Errorf("failed to retrieve release notes: %w", err)
			}

			for _, relNote := range releaseNotesCache {
				shortVersion := relNote.ShortVersionFormat()
				if strings.HasPrefix(shortVersion, version) && relNote.BuildNumber() == build {
					return shortVersion, nil
				}
			}
		} else {
			return "", errors.New("expected a string version")
		}
	}
	return "", errors.New("input version not provided")
}
