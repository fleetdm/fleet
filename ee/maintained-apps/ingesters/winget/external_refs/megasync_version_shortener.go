package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// MEGAsyncVersionShortener transforms the MEGAsync version from the winget
// manifest format to the version its NSIS installer registers in Windows
// Programs.
//
// Winget reports: "6.7.0.3" (4-part version with a build number)
// Installer registers: "6.7.0" (3-part version)
//
// osquery's version_compare sorts "6.7.0" below "6.7.0.3", so without this the
// patch policy fails on hosts that already have the current version.
func MEGAsyncVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 3:
		// winget occasionally publishes the 3-part version (6.2.2).
		return app, nil
	case 4:
		app.Version = strings.Join(parts[:3], ".")
		return app, nil
	default:
		return app, fmt.Errorf("expected MEGAsync version to have 3 or 4 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
