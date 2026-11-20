package externalrefs

import (
	"fmt"
	"strconv"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// TemurinMSIVersion transforms an Eclipse Temurin (11+) version from the winget
// manifest format to the version its MSI registers in Windows Programs.
//
// Point releases (jdk-17.0.21.1+2) encode the patch and build in the fourth part
// differently:
//
// Winget reports: "17.0.21.102" (patch × 100 + build)
// Installer registers: "17.0.21.10002" (patch × 10,000 + build, Adoptium's
// formMSIProductVersion in adoptium/jenkins-helper)
//
// Regular releases ("17.0.21.9") are the same in both. Without this, Fleet
// shows a version that never appears in inventory.
func TemurinMSIVersion(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	if len(parts) != 4 {
		return app, fmt.Errorf("expected Temurin version to have 4 parts but found %d parts in '%s'", len(parts), app.Version)
	}
	rev, err := strconv.Atoi(parts[3])
	if err != nil || rev < 0 {
		return app, fmt.Errorf("expected numeric fourth part in Temurin version '%s'", app.Version)
	}
	if rev < 100 {
		return app, nil
	}
	parts[3] = strconv.Itoa(rev/100*10000 + rev%100)
	app.Version = strings.Join(parts, ".")
	return app, nil
}
