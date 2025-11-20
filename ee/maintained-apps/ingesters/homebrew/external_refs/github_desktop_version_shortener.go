package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// GitHubDesktopVersionShortener strips the git commit hash from GitHub Desktop versions.
// Homebrew provides versions like "3.5.4-9dfb8d8d", but osquery reports "3.5.4".
// This transformer extracts just the semantic version part.
func GitHubDesktopVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Split on '-' and take the first part (the semantic version)
	parts := strings.Split(homebrewVersion, "-")
	if len(parts) > 0 && parts[0] != "" {
		app.Version = parts[0]
	} else {
		return app, fmt.Errorf("Expected GitHub Desktop version to contain a semantic version but found '%s'", homebrewVersion)
	}

	return app, nil
}
