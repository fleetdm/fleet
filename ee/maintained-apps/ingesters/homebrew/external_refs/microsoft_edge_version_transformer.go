package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// MicrosoftEdgeVersionTransformer extracts the version part from Homebrew's comma-separated format.
// Homebrew reports versions like "142.0.3595.94,9d2b7e5f-8c6f-4661-9c90-afadc2befce6" where
// the comma separates the version from a UUID. This transformer extracts just the version part.
func MicrosoftEdgeVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Split on comma and take the first part (the version)
	parts := strings.Split(app.Version, ",")
	if len(parts) > 0 && parts[0] != "" {
		app.Version = parts[0]
	}
	return app, nil
}
