package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// VSCodeUniversalInstaller rewrites Homebrew's arm64-only VS Code download URL
// to Microsoft's universal build, so the FMA installs natively on both Intel
// and Apple silicon Macs.
//
// Homebrew exposes only architecture-specific builds and no universal variant:
//
//	https://update.code.visualstudio.com/<version>/darwin-arm64/stable  (arm64)
//	https://update.code.visualstudio.com/<version>/darwin/stable        (intel)
//
// Microsoft serves the universal (x86_64 + arm64) build at a path Homebrew
// doesn't reference:
//
//	https://update.code.visualstudio.com/<version>/darwin-universal/stable
//
// /darwin/ is the Intel build, not universal.
//
// Since the URL (and therefore the artifact) changes, the Homebrew SHA256 no
// longer applies; set it to "no_check" as the other installer-URL overrides do.
func VSCodeUniversalInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.InstallerURL = strings.Replace(app.InstallerURL, "/darwin-arm64/", "/darwin-universal/", 1)
	app.SHA256 = "no_check"

	return app, nil
}
