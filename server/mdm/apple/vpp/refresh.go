package vpp

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/itunes"
)

// RefreshVersions updatest the LatestVersion fields for the VPP apps stored in Fleet.
func RefreshVersions(ctx context.Context, ds fleet.Datastore) error {
	apps, err := ds.GetAllVPPApps(ctx)
	if err != nil {
		return err
	}

	var adamIDs []string
	appsByAdamID := make(map[string]*fleet.VPPApp)
	for _, app := range apps {
		adamIDs = append(adamIDs, app.AdamID)
		appsByAdamID[app.AdamID] = app
	}

	meta, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return err
	}

	var appsToUpdate []*fleet.VPPApp
	for _, adamID := range adamIDs {
		if m, ok := meta[adamID]; ok {
			if m.Version != appsByAdamID[adamID].LatestVersion {
				appsByAdamID[adamID].LatestVersion = m.Version
				appsToUpdate = append(appsToUpdate, appsByAdamID[adamID])
			}
		}
	}

	if err := ds.InsertVPPApps(ctx, appsToUpdate); err != nil {
		return err
	}

	return nil
}
