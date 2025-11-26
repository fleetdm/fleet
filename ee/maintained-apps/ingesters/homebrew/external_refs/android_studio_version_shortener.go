package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// AndroidStudioVersionShortener trims the version to 2 parts.
// Homebrew version format: "2025.2.1.8" (4 parts)
// osquery bundle_short_version: "2025.2" (2 parts only!)
// This extracts the first two parts to match osquery's bundle_short_version.
// Note: This means patch versions (2025.2.1.x vs 2025.2.2.x) will not be distinguishable.
func AndroidStudioVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Split on dots and take the first two parts
	parts := strings.Split(homebrewVersion, ".")
	if len(parts) >= 2 {
		app.Version = strings.Join(parts[:2], ".")
	} else {
		return app, fmt.Errorf("Expected Android Studio version to have format X.Y.Z.W but found '%s'", homebrewVersion)
	}

	return app, nil
}
