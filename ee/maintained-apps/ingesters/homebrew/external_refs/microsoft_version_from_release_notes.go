package externalrefs

import (
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
//
// If the exact build number is not found in the release notes (e.g., because the
// release notes page hasn't been updated yet for a newly published build), the
// function falls back to the base version without the build number (e.g., "16.95"
// instead of "16.95.25032931"). This prevents a perpetual "update available" loop
// that would otherwise occur because osquery reports the short version
// (e.g., "16.95.3") while the manifest stores the raw build string, causing
// compareVersions to always flag the installed version as older.
func MicrosoftVersionFromReleaseNotes(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version
	versionParts := strings.Split(homebrewVersion, ".") // homebrew version format is like "16.95.25032931"

	// If the version doesn't have at least major.minor.build segments, there is no
	// build number to extract and look up; return the version unchanged.
	if len(versionParts) < 3 {
		return app, nil
	}

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

	// No exact match found in the release notes for this build number. Fall back to
	// the base version (e.g., "16.95" from "16.95.25032931") so that the installed
	// short version (e.g., "16.95.3") is not falsely flagged as older.
	app.Version = version
	return app, nil
}
