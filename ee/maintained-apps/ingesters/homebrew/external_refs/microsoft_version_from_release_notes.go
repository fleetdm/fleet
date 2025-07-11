package externalrefs

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	macoffice "github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
)

var (
	releaseNotesCache macoffice.ReleaseNotes
	cacheOnce         sync.Once
)

// MicrosoftVersionFromReleaseNotes modifies the app version
// Gets the short version from the release notes given a Homebrew version.
// Homebrew version "16.95.25032931"
// Release notes version "Version 16.95.3 (Build 25032931)"
// For this example it would change the app version to "16.95.3"
func MicrosoftVersionFromReleaseNotes(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version
	versionParts := strings.Split(homebrewVersion, ".")              // homebrew version format is like "16.95.25032931"
	version := strings.Join(versionParts[:len(versionParts)-1], ".") // Extract version without the build number
	build := versionParts[len(versionParts)-1]                       // Extract the build number

	var err error
	cacheOnce.Do(func() {
		releaseNotesCache, err = macoffice.GetReleaseNotes(true)
	})
	if err != nil {
		cacheOnce = sync.Once{}
		return app, fmt.Errorf("failed to retrieve release notes: %w", err)
	}

	for _, relNote := range releaseNotesCache {
		shortVersion := relNote.ShortVersionFormat()
		if strings.HasPrefix(shortVersion, version) && relNote.BuildNumber() == build {
			app.Version = shortVersion
			return app, nil
		}
	}

	return app, fmt.Errorf("no matching version found in release notes for %s", homebrewVersion)
}
