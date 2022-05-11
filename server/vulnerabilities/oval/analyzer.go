package oval

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	versions *fleet.OSVersions,
	vulnPath string,
) error {
	for _, s := range SupportedPlatforms {
		for _, version := range versions.OSVersions {
			if s != version.Platform {
				continue
			}

			// Load oval definitions
			// Iterate over host id for platform
			// Get software for host id
		}
	}

	return nil
}
