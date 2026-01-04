package vpp

import (
	"context"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
)

// RefreshVersions updatest the LatestVersion fields for the VPP apps stored in Fleet.
func RefreshVersions(ctx context.Context, ds fleet.Datastore) error {
	apps, err := ds.GetAllVPPApps(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting all VPP apps")
	}

	if len(apps) == 0 {
		// nothing to do
		return nil
	}

	// We use a map for applications that share the same Adam ID for their iOS/iPadOS/macOS apps.
	appsByAdamID := make(map[string][]*fleet.VPPApp) // Adam ID -> one, two, or three `*fleet.VPPApp`s.
	// Gathering adamIDs on a set to deduplicate them and send them to iTunes.
	adamIDs := make(map[string]struct{})
	for _, app := range apps {
		adamIDs[app.AdamID] = struct{}{}
		appsByAdamID[app.AdamID] = append(appsByAdamID[app.AdamID], app)

	}
	adamIDsToQueryITunes := slices.Collect(maps.Keys(adamIDs))

	meta, err := apple_apps.GetMetadata(adamIDsToQueryITunes)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting VPP app metadata from iTunes API")
	}

	var appsToUpdate []*fleet.VPPApp
	for _, adamID := range adamIDsToQueryITunes {
		if m, ok := meta[adamID]; ok {
			// Iterate all platforms for the Adam ID (iOS/iPadOS/macOS).
			for _, app := range appsByAdamID[adamID] {
				if m.Version != app.LatestVersion {
					app.LatestVersion = m.Version
					appsToUpdate = append(appsToUpdate, app)
				}
			}
		}
	}

	if len(appsToUpdate) == 0 {
		// nothing to do
		return nil
	}

	if err := ds.InsertVPPApps(ctx, appsToUpdate); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting VPP apps with new versions")
	}

	return nil
}
