package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
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

	vppTokenRawBytes, err := base64.StdEncoding.DecodeString(vppTokenData.Token)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decoding raw vpp token data")
	}

	var vppTokenRaw fleet.VPPTokenRaw

	if err := json.Unmarshal(vppTokenRawBytes, &vppTokenRaw); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshaling raw vpp token data")
	}

	exp, err := time.Parse("2006-01-02T15:04:05Z0700", vppTokenRaw.ExpDate)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parsing vpp token expiration date")
	}

	if time.Now().After(exp) {
		return "", ctxerr.Errorf(ctx, "vpp token expired on %s", exp.String())
	}

	return vppTokenData.Token, nil
}

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []fleet.VPPBatchPayload, dryRun bool) error {
	if teamName == "" {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "must not be empty"))
	}

	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return err
	}

	team, err := svc.ds.TeamByName(ctx, teamName)
	if err != nil {
		// If this is a dry run, the team may not have been created yet
		if dryRun && fleet.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &team.ID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "validating authorization")
	}

	var adamIDs []string

	// Don't check for token if we're only disassociating assets
	if len(payloads) > 0 {
		token, err := svc.getVPPToken(ctx)
		if err != nil {
			return fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "could not retrieve vpp token"), http.StatusUnprocessableEntity)
		}

		for _, payload := range payloads {
			adamIDs = append(adamIDs, payload.AppStoreID)
		}

		var missingAssets []string

		assets, err := vpp.GetAssets(token, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unable to retrieve assets")
		}

		assetMap := map[string]struct{}{}
		for _, asset := range assets {
			assetMap[asset.AdamID] = struct{}{}
		}

		for _, adamID := range adamIDs {
			if _, ok := assetMap[adamID]; !ok {
				missingAssets = append(missingAssets, adamID)
			}
		}

		if len(missingAssets) != 0 {
			reqErr := ctxerr.Errorf(ctx, "requested app not available on vpp account: %s", strings.Join(missingAssets, ","))
			return fleet.NewUserMessageError(reqErr, http.StatusUnprocessableEntity)
		}
	}

	if !dryRun {
		apps, err := getVPPAppsMetadata(ctx, adamIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "refreshing VPP app metadata")
		}

		if err := svc.ds.BatchInsertVPPApps(ctx, apps); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting vpp app metadata")
		}

		if err := svc.ds.SetTeamVPPApps(ctx, &team.ID, adamIDs); err != nil {
			return fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "set team vpp assets"), http.StatusInternalServerError)
		}
	}

	return nil
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
	var appsToUpdate []*fleet.VPPApp
	for _, a := range assets {
		m, ok := assetMetadata[a.AdamID]
		if !ok {
			// Then this adam_id belongs to a non-desktop app.
			continue
		}

		app := &fleet.VPPApp{
			AdamID:           a.AdamID,
			BundleIdentifier: m.BundleID,
			IconURL:          m.ArtworkURL,
			Name:             m.TrackName,
			LatestVersion:    m.Version,
		}

		if _, ok := assignedApps[a.AdamID]; ok {
			// Then this is already assigned, so filter it out.
			appsToUpdate = append(appsToUpdate, app)
			continue
		}

		apps = append(apps, app)
	}

	if len(appsToUpdate) > 0 {
		if err := svc.ds.BatchInsertVPPApps(ctx, appsToUpdate); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "updating existing VPP apps")
		}
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
		BundleIdentifier: assetMD.BundleID,
		IconURL:          assetMD.ArtworkURL,
		Name:             assetMD.TrackName,
		LatestVersion:    assetMD.Version,
	}
	if _, err := svc.ds.InsertVPPAppWithTeam(ctx, app, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "writing VPP app to db")
	}

	act := fleet.ActivityAddedAppStoreApp{
		AppStoreID:    app.AdamID,
		TeamName:      &teamName,
		SoftwareTitle: app.Name,
		TeamID:        teamID,
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for add app store app")
	}

	return nil
}

func getVPPAppsMetadata(ctx context.Context, adamIDs []string) ([]*fleet.VPPApp, error) {
	var apps []*fleet.VPPApp

	assetMetatada, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "desktopSoftware"})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	for adamID, metadata := range assetMetatada {
		app := &fleet.VPPApp{
			AdamID:           adamID,
			BundleIdentifier: metadata.BundleID,
			IconURL:          metadata.ArtworkURL,
			Name:             metadata.TrackName,
			LatestVersion:    metadata.Version,
		}

		apps = append(apps, app)
	}

	return apps, nil
}
