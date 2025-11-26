package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func CitrixWorkspaceVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	parts := strings.Split(homebrewVersion, ".")
	if len(parts) == 4 {
		// Citrix Workspace version format: XX.XX.XX.XX
		// We need the first 3 segments: XX.XX.XX
		app.Version = strings.Join(parts[:3], ".")
	} else {
		return app, fmt.Errorf("Expected Citrix Workspace version to match XX.XX.XX.XX but found '%s'", homebrewVersion)
	}

	return app, nil
}
