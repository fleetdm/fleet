package externalrefs

import (
	"fmt"
	"strconv"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func BraveVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	var modifiedVersion string
	homebrewVersion := app.Version
	// Input format: 1.79.123.0

	// drop lagging .0
	if strings.HasSuffix(homebrewVersion, ".0") {
		modifiedVersion = homebrewVersion[:len(homebrewVersion)-2]
	} else {
		return app, fmt.Errorf("Expected Brave version to end with '.0' but found '%s'", homebrewVersion)
	}

	// add 58 to second value
	parts := strings.Split(modifiedVersion, ".")
	if len(parts) == 3 {
		if second, err := strconv.Atoi(parts[1]); err == nil {
			parts = append([]string{fmt.Sprintf("%d", second+58)}, parts...)
		} else {
			return app, fmt.Errorf("Failed to parse '%s' of Brave version '%s': %v", parts[1], homebrewVersion, err)
		}
		// Output format: 137.1.79.123
		app.Version = strings.Join(parts, ".")
	} else {
		return app, fmt.Errorf("Expected Brave version to have four parts but found '%s'", homebrewVersion)
	}

	return app, nil
}
