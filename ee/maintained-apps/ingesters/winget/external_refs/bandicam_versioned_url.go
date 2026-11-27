package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

const bandicamRollingURL = "https://dl.bandicam.com/bdcamsetup.exe"

// BandicamVersionedURL swaps Bandicam's rolling download URL for the versioned
// copy the vendor publishes next to it.
//
// Winget reports: https://dl.bandicam.com/bdcamsetup.exe (always the newest build)
// Vendor also serves: https://dl.bandicam.com/old/BDCAMSETUP_27_0_7_2687.EXE
//
// The rolling URL serves the next build as soon as the vendor ships it, which
// would break the pinned SHA-256 until winget and Fleet catch up.
func BandicamVersionedURL(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if !strings.EqualFold(app.InstallerURL, bandicamRollingURL) {
		return app, nil
	}

	parts := strings.Split(app.Version, ".")
	if len(parts) != 4 {
		return app, fmt.Errorf("expected Bandicam version to have 4 parts but found %d parts in '%s'", len(parts), app.Version)
	}
	for _, part := range parts {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return app, fmt.Errorf("expected Bandicam version parts to be numeric but found '%s'", app.Version)
		}
	}

	app.InstallerURL = "https://dl.bandicam.com/old/BDCAMSETUP_" + strings.Join(parts, "_") + ".EXE"
	return app, nil
}
