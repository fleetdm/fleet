package vpp

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
)

// RefreshVersions updatest the LatestVersion fields for the VPP apps stored in Fleet.
func RefreshVersions(ctx context.Context, ds fleet.Datastore, vppAppsConfig apple_apps.Config) error {
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
	// Group adamIDs by the region their metadata was last resolved against, so we can target the
	// correct storefront per app. Legacy rows (empty region) are queried from the default region.
	adamIDsByRegion := make(map[string]map[string]struct{})
	for _, app := range apps {
		region := app.MetadataRegion
		if region == "" {
			region = apple_apps.DefaultMetadataRegion
		}
		if _, ok := adamIDsByRegion[region]; !ok {
			adamIDsByRegion[region] = make(map[string]struct{})
		}
		adamIDsByRegion[region][app.AdamID] = struct{}{}
		appsByAdamID[app.AdamID] = append(appsByAdamID[app.AdamID], app)
	}

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
		for region, adamIDSet := range adamIDsByRegion {
			if len(adamIDSet) == 0 {
				continue
			}
			adamIDs := make([]string, 0, len(adamIDSet))
			for id := range adamIDSet {
				adamIDs = append(adamIDs, id)
			}
			meta, err := apple_apps.GetMetadata(adamIDs, region, vppToken.Token, vppAppsConfig)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting VPP app metadata from Apple API")
			}

			for k, v := range meta {
				retrievedApps[k] = apple_apps.ToVPPApps(v)
				// Stop searching for this adamID once any token has returned it.
				delete(adamIDSet, k)
			}
		}
	}

	for adamID, storedApps := range appsByAdamID {
		retrievedByPlatform, ok := retrievedApps[adamID]
		if !ok {
			continue
		}
		for _, app := range storedApps {
			if current, ok := retrievedByPlatform[app.Platform]; ok && current.LatestVersion != app.LatestVersion {
				app.LatestVersion = current.LatestVersion
				appsToUpdate = append(appsToUpdate, app)
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
