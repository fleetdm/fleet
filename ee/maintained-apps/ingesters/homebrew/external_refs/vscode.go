package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// VSCodeUniversalInstaller rewrites Homebrew's Apple-silicon-only download URL
// to the architecture-independent "universal" build, so the FMA installs on
// both Intel and Apple silicon Macs.
//
// Homebrew exposes the arm64 build as the cask's url:
//
//	https://update.code.visualstudio.com/<version>/darwin-arm64/stable
//
// Microsoft serves the universal build at:
//
//	https://update.code.visualstudio.com/<version>/darwin-universal/stable
//
// Since the URL (and therefore the artifact) changes, the Homebrew SHA256 no
// longer applies; set it to "no_check" as the other installer-URL overrides do.
func VSCodeUniversalInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.InstallerURL = strings.Replace(app.InstallerURL, "/darwin-arm64/", "/darwin-universal/", 1)
	app.SHA256 = "no_check"

	return app, nil
}
