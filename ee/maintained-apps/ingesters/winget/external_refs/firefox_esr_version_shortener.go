package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// FirefoxESRVersionShortener transforms the version of the first release of a
// Firefox ESR major from the winget manifest format to the version the
// installer registers in Windows Programs.
//
// Winget reports: "153.0.0"
// Installer registers: "153.0" (Mozilla's version for the first release of a
// major; later releases keep three parts: 153.4.0 registers "153.4.0")
//
// osquery's version_compare sorts "153.0" below "153.0.0", so without this the
// patch policy fails on hosts that already have the current version.
func FirefoxESRVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 2:
		return app, nil
	case 3:
		if parts[1] == "0" && parts[2] == "0" {
			app.Version = strings.Join(parts[:2], ".")
		}
		return app, nil
	default:
		return app, fmt.Errorf("expected Firefox ESR version to have 2 or 3 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
