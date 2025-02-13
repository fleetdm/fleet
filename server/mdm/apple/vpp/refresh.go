package vpp

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/itunes"
)

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

	for _, adamID := range adamIDs {
		if m, ok := meta[adamID]; ok {
			appsByAdamID[adamID].LatestVersion = m.Version
		}
	}

	if err := ds.InsertVPPApps(ctx, apps); err != nil {
		return err
	}

	return nil
}
