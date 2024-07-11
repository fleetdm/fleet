package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/itunes"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
)

func (svc *Service) getVPPToken(ctx context.Context) (string, error) {
	configMap, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetVPPToken})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "fetching vpp token")
	}

	var vppTokenData fleet.VPPTokenData
	if err := json.Unmarshal(configMap[fleet.MDMAssetVPPToken].Value, &vppTokenData); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshaling VPP token data")
	}

	return base64.StdEncoding.EncodeToString([]byte(vppTokenData.Token)), nil
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*fleet.VPPApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vppToken, err := svc.getVPPToken(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(vppToken, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching Apple VPP assets")
	}

	if len(assets) == 0 {
		return []*fleet.VPPApp{}, nil
	}

	var adamIDs []string
	for _, a := range assets {
		adamIDs = append(adamIDs, a.AdamID)
	}

	assetMetadata, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "desktopSoftware"})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	assignedApps, err := svc.ds.GetAssignedVPPApps(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving assigned VPP apps")
	}

	var apps []*fleet.VPPApp
	for _, a := range assets {
		m, ok := assetMetadata[a.AdamID]
		if !ok {
			// Then this adam_id belongs to a non-desktop app.
			continue
		}

		if _, ok := assignedApps[a.AdamID]; ok {
			// Then this is already assigned, so filter it out.
			continue
		}

		apps = append(apps, &fleet.VPPApp{
			AdamID:           a.AdamID,
			AvailableCount:   a.AvailableCount,
			BundleIdentifier: m.BundleID,
			IconURL:          m.ArtworkURL,
			Name:             m.TrackName,
			LatestVersion:    m.Version,
		})
	}

	return apps, nil
}

func (svc *Service) AddAppStoreApp(ctx context.Context, teamID *uint, adamID string) error {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	var teamName string
	if teamID != nil {
		tm, err := svc.ds.Team(ctx, *teamID)
		if fleet.IsNotFound(err) {
			return fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if team exists")
		}

		teamName = tm.Name
	}

	vppToken, err := svc.getVPPToken(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(vppToken, &vpp.AssetFilter{AdamID: adamID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP asset")
	}

	if len(assets) == 0 {
		return ctxerr.New(ctx, fmt.Sprintf("Error: Couldn't add software. %s isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.", adamID))
	}

	asset := assets[0]

	assetMetadata, err := itunes.GetAssetMetadata([]string{asset.AdamID}, &itunes.AssetMetadataFilter{Entity: "desktopSoftware"})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	assetMD := assetMetadata[asset.AdamID]

	// Check if we've already added an installer for this app
	exists, err := svc.ds.UploadedSoftwareExists(ctx, assetMD.BundleID, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking existence of VPP app installer")
	}

	if exists {
		return ctxerr.New(ctx, fmt.Sprintf("Error: Couldn't add software. %s already has software available for install on the %s team.", assetMD.TrackName, teamName))
	}

	app := &fleet.VPPApp{
		AdamID:           asset.AdamID,
		AvailableCount:   asset.AvailableCount,
		BundleIdentifier: assetMD.BundleID,
		IconURL:          assetMD.ArtworkURL,
		Name:             assetMD.TrackName,
	}
	if err := svc.ds.InsertVPPAppWithTeam(ctx, app, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "writing VPP app to db")
	}

	return nil
}
