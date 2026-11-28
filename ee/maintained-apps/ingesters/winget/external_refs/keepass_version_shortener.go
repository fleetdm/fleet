package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// KeePassVersionShortener transforms the KeePass version from the winget
// manifest format to the version its installer registers in Windows Programs.
//
// Winget reports: "2.62.0"
// Installer registers: "2.62" (KeePass omits a zero patch part; 2.61.1
// registers "2.61.1")
//
// osquery's version_compare sorts "2.62" below "2.62.0", so without this the
// patch policy fails on hosts that already have the current version.
func KeePassVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 2:
		return app, nil
	case 3:
		if parts[2] == "0" {
			app.Version = strings.Join(parts[:2], ".")
		}
		return app, nil
	default:
		return app, fmt.Errorf("expected KeePass version to have 2 or 3 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
