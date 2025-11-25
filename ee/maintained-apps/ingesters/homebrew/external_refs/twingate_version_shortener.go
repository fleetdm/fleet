package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// TwingateVersionShortener extracts the bundle_short_version from Homebrew's version format.
// Homebrew version format: "2025.288.20108"
// osquery bundle_short_version: "2025.288"
// osquery bundle_version: "120108"
// This extracts the first two parts (before the last dot) to match osquery's bundle_short_version.
func TwingateVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Split on dots and take the first two parts (bundle_short_version)
	parts := strings.Split(homebrewVersion, ".")
	if len(parts) >= 2 {
		app.Version = strings.Join(parts[:2], ".")
	} else {
		return app, fmt.Errorf("Expected Twingate version to have format X.Y.Z but found '%s'", homebrewVersion)
	}

	return app, nil
}

