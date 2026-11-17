package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// AOMEIBackupperVersionShortener transforms the AOMEI Backupper Standard
// version from the winget manifest format to the version its Inno Setup
// installer registers in Windows Programs.
//
// Winget reports: "8.5.0.0" (4-part version)
// Installer registers: "8.5.0" (3-part version)
//
// osquery's version_compare sorts "8.5.0" below "8.5.0.0", so without this the
// patch policy fails on hosts that already have the current version.
func AOMEIBackupperVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 3:
		return app, nil
	case 4:
		// Every 4-part winget version so far ends in ".0"; a non-zero fourth
		// part may be registered differently, so it needs a fresh look.
		if parts[3] != "0" {
			return app, fmt.Errorf("expected AOMEI Backupper version to end in '.0' but found '%s'", app.Version)
		}
		app.Version = strings.Join(parts[:3], ".")
		return app, nil
	default:
		return app, fmt.Errorf("expected AOMEI Backupper version to have 3 or 4 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
