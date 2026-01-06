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
	// Gathering adamIDs on a set to deduplicate them and send them to Apple's Apps & Books API.
	adamIDs := make(map[string]struct{})
	for _, app := range apps {
		adamIDs[app.AdamID] = struct{}{}
		appsByAdamID[app.AdamID] = append(appsByAdamID[app.AdamID], app)

	}
	adamIDsToQuery := slices.Collect(maps.Keys(adamIDs))

	// in a multi-VPP-token environment, custom apps may be visible to one VPP token but not another;
	// if you request apps that aren't visible from the Apple API the requests will take longer but
	// will still return, so we can iterate through VPP tokens on hand until we have all apps enumerated
	// to get the latest versions for each.
	vppTokens, err := ds.ListVPPTokens(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting all VPP tokens")
	}

	retrievedApps := make(map[string]map[fleet.InstallableDevicePlatform]fleet.VPPApp)
	var appsToUpdate []*fleet.VPPApp

	for _, vppToken := range vppTokens {
		meta, err := apple_apps.GetMetadata(adamIDsToQuery, vppToken.Token, apple_apps.GetAppMetadataBearerToken(ds))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting VPP app metadata from Apple API")
		}

		for k, v := range meta {
			retrievedApps[k] = apple_apps.ToVPPApps(v)
		}

		// we found all apps, either because they are all public or because we've iterated over enough VPP keys
		// to retrieve all custom apps; we don't need to request with any more apps
		if len(retrievedApps) >= len(adamIDsToQuery) {
			continue
		}
	}

	for _, adamID := range adamIDsToQuery {
		if retrievedByPlatform, ok := retrievedApps[adamID]; ok {
			for _, app := range appsByAdamID[adamID] {
				if current, ok := retrievedByPlatform[app.Platform]; ok && current.LatestVersion != app.LatestVersion {
					app.LatestVersion = current.LatestVersion
					appsToUpdate = append(appsToUpdate, app)
				}
			}
		}
	}

	if len(appsToUpdate) == 0 { // nothing to do
		return nil
	}

	if err := ds.InsertVPPApps(ctx, appsToUpdate); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting VPP apps with new versions")
	}

	return nil
}
