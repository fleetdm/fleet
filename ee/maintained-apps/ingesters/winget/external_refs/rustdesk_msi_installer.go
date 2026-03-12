package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// RustDeskMSIInstaller swaps the RustDesk installer from EXE to MSI.
//
// RustDesk's EXE installer uses a custom --silent-install flag that does not
// reliably install in CI/MDM environments. The MSI installer uses standard
// msiexec /quiet and is more reliable for fleet-managed installs.
//
// The MSI is available at the same GitHub release URL, just with a .msi extension.
func RustDeskMSIInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if !strings.HasSuffix(app.InstallerURL, ".exe") {
		return app, fmt.Errorf("expected RustDesk installer URL to end in .exe, got: %s", app.InstallerURL)
	}

	// Swap EXE URL to MSI URL (same release, different package format)
	app.InstallerURL = strings.TrimSuffix(app.InstallerURL, ".exe") + ".msi"

	// MSI hash is not in the winget manifest, so skip the check
	app.SHA256 = "no_check"

	return app, nil
}
