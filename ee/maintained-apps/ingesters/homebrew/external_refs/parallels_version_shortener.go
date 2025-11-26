package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// ParallelsVersionShortener extracts the bundle_short_version from Homebrew's version format.
// Homebrew version format: "26.1.2-57293"
// osquery bundle_short_version: "26.1.2"
// osquery bundle_version: "57293"
// This extracts the part before the dash to match osquery's bundle_short_version.
func ParallelsVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Split on dash and take the first part (bundle_short_version)
	parts := strings.Split(homebrewVersion, "-")
	if len(parts) >= 1 && parts[0] != "" {
		app.Version = parts[0]
	} else {
		return app, fmt.Errorf("Expected Parallels version to have format X.Y.Z-BUILD but found '%s'", homebrewVersion)
	}

	return app, nil
}
