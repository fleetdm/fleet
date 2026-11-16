package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// AnyBurnVersionShortener transforms the AnyBurn version from the winget
// manifest format to the version its NSIS installer registers in Windows
// Programs.
//
// Winget reports: "6.9.0.0" (the winget bot's 4-part version)
// Installer registers: "6.9" (DisplayVersion built from VersionMajor.VersionMinor)
//
// osquery's version_compare sorts "6.9" below "6.9.0.0", so without this the
// patch policy fails on hosts that already have the current version.
func AnyBurnVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	if len(parts) < 2 {
		return app, fmt.Errorf("expected AnyBurn version to have at least 2 parts but found %d parts in '%s'", len(parts), app.Version)
	}
	// The registry only carries major.minor, so a non-zero third or fourth part
	// can't be compared against it.
	for _, p := range parts[2:] {
		if p != "0" {
			return app, fmt.Errorf("expected AnyBurn version to be major.minor with only zero parts after it but found '%s'", app.Version)
		}
	}

	app.Version = strings.Join(parts[:2], ".")

	return app, nil
}
