package externalrefs

import (
	"fmt"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// wingetReportedDatadogAgentVersion is the PackageVersion winget's manifest
// reports for the pinned Datadog Agent MSI this app ships.
const wingetReportedDatadogAgentVersion = "7.81.2.1"

// msiActualDatadogAgentVersion is the installer's real MSI Property table
// ProductVersion (verified with `msiinfo export ddagent-cli-7.81.2.msi
// Property`), i.e. what osquery's programs.version actually reports once the
// MSI is installed.
const msiActualDatadogAgentVersion = "7.81.2.0"

// DatadogAgentVersionFixer corrects the winget-reported package version to
// match what the MSI actually registers in Windows Programs.
//
// Winget reports: "7.81.2.1"
// MSI ProductVersion (and what osquery finds post-install): "7.81.2.0"
//
// The installer is pinned (fixed URL + SHA256, not a rolling "latest" build),
// so this is a one-off metadata mismatch in the winget catalog for this
// release rather than version drift over time.
func DatadogAgentVersionFixer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if app.Version != wingetReportedDatadogAgentVersion {
		return app, fmt.Errorf(
			"expected Datadog Agent winget version to be '%s' but found '%s'; re-verify the MSI's real ProductVersion and update this fixer (or remove it if winget has since corrected the drift)",
			wingetReportedDatadogAgentVersion, app.Version,
		)
	}

	app.Version = msiActualDatadogAgentVersion
	return app, nil
}
