package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// DatadogAgentVersionNormalizer transforms the Datadog Agent version from the
// winget manifest format to the version its MSI registers in Windows Programs.
//
// Winget reports: "7.85.2.1" (the fourth part is Datadog's package release)
// Installer registers: "7.85.2.0" (MSI ProductVersion)
//
// osquery's version_compare sorts "7.85.2.0" below "7.85.2.1", so without this
// the patch policy fails on hosts that already have the current version.
func DatadogAgentVersionNormalizer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 3:
		app.Version += ".0"
		return app, nil
	case 4:
		parts[3] = "0"
		app.Version = strings.Join(parts, ".")
		return app, nil
	default:
		return app, fmt.Errorf("expected Datadog Agent version to have 3 or 4 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
