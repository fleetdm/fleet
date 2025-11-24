package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// OnePasswordVersionShortener transforms 1Password version from winget manifest format
// to the version that actually gets registered in Windows Programs.
//
// Winget reports: "8.11.18.36" (4-part version)
// MSI registers: "8.11.18" (3-part version)
//
// This transformer strips the 4th version part to match what osquery will find.
func OnePasswordVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	wingetVersion := app.Version

	// Split version into parts
	parts := strings.Split(wingetVersion, ".")

	// Expected format: X.Y.Z.W (4 parts)
	if len(parts) != 4 {
		return app, fmt.Errorf("expected 1Password version to have 4 parts but found %d parts in '%s'", len(parts), wingetVersion)
	}

	// Trim to first 3 parts: X.Y.Z
	app.Version = strings.Join(parts[:3], ".")

	return app, nil
}
