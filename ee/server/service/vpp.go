package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/itunes"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
)

// getVPPToken returns the base64 encoded VPP token, ready for use in requests to Apple's VPP API.
// It returns an error if the token is expired.
func (svc *Service) getVPPToken(ctx context.Context, teamID *uint) (string, error) {
	token, err := svc.ds.GetVPPTokenByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fleet.NewUserMessageError(errors.New("No available VPP Token"), http.StatusUnprocessableEntity)
		}
		return "", ctxerr.Wrap(ctx, err, "fetching vpp token")
	}

	if time.Now().After(token.RenewDate) {
		return "", fleet.NewUserMessageError(errors.New("Couldn't install. VPP token expired."), http.StatusUnprocessableEntity)
	}

	return token.Token, nil
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

	// Adding VPP apps will add them to all available platforms per decision:
	// https://github.com/fleetdm/fleet/issues/19447#issuecomment-2256598681
	// The code is already here to support individual platforms, so we can easily enable it later.

	payloadsWithPlatform := make([]fleet.VPPBatchPayloadWithPlatform, 0, len(payloads))
	for _, payload := range payloads {
		// Currently only macOS is supported for self-service. Don't
		// import vpp apps as self-service for ios or ipados
		payloadsWithPlatform = append(payloadsWithPlatform, []fleet.VPPBatchPayloadWithPlatform{{
			AppStoreID:  payload.AppStoreID,
			SelfService: false,
			Platform:    fleet.IOSPlatform,
		}, {
			AppStoreID:  payload.AppStoreID,
			SelfService: false,
			Platform:    fleet.IPadOSPlatform,
		}, {
			AppStoreID:  payload.AppStoreID,
			SelfService: payload.SelfService,
			Platform:    fleet.MacOSPlatform,
		}}...)
	}

	var vppAppTeams []fleet.VPPAppTeam
	// Don't check for token if we're only disassociating assets
	if len(payloads) > 0 {
		token, err := svc.getVPPToken(ctx, &team.ID)
		if err != nil {
			return fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "could not retrieve vpp token"), http.StatusUnprocessableEntity)
		}

		for _, payload := range payloadsWithPlatform {
			if payload.Platform == "" {
				payload.Platform = fleet.MacOSPlatform
			}
			if payload.Platform != fleet.IOSPlatform && payload.Platform != fleet.IPadOSPlatform && payload.Platform != fleet.MacOSPlatform {
				return fleet.NewInvalidArgumentError("app_store_apps.platform",
					fmt.Sprintf("platform must be one of '%s', '%s', or '%s", fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform))
			}
			vppAppTeams = append(vppAppTeams, fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: payload.AppStoreID, Platform: payload.Platform}, SelfService: payload.SelfService})
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

		for _, vppAppID := range vppAppTeams {
			if _, ok := assetMap[vppAppID.AdamID]; !ok {
				missingAssets = append(missingAssets, vppAppID.AdamID)
			}
		}

		if len(missingAssets) != 0 {
			reqErr := ctxerr.Errorf(ctx, "requested app not available on vpp account: %s", strings.Join(missingAssets, ","))
			return fleet.NewUserMessageError(reqErr, http.StatusUnprocessableEntity)
		}
	}

	if !dryRun {
		if len(vppAppTeams) > 0 {
			apps, err := getVPPAppsMetadata(ctx, vppAppTeams)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "refreshing VPP app metadata")
			}
			if len(apps) == 0 {
				return fleet.NewInvalidArgumentError("app_store_apps",
					"no valid apps found matching the provided app store IDs and platforms")
			}

			if err := svc.ds.BatchInsertVPPApps(ctx, apps); err != nil {
				return ctxerr.Wrap(ctx, err, "inserting vpp app metadata")
			}
			// Filter out the apps with invalid platforms
			if len(apps) != len(vppAppTeams) {
				vppAppTeams = make([]fleet.VPPAppTeam, 0, len(apps))
				for _, app := range apps {
					vppAppTeams = append(vppAppTeams, app.VPPAppTeam)
				}
			}
		}

		if err := svc.ds.SetTeamVPPApps(ctx, &team.ID, vppAppTeams); err != nil {
			return fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "set team vpp assets"), http.StatusInternalServerError)
		}
	}

	return nil
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*fleet.VPPApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vppToken, err := svc.getVPPToken(ctx, teamID)
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

	assetMetadata, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "software"})
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
			// Then this adam_id is not a VPP software entity, so skip it.
			continue
		}

		platforms := getPlatformsFromSupportedDevices(m.SupportedDevices)

		for platform := range platforms {
			vppAppID := fleet.VPPAppID{
				AdamID:   a.AdamID,
				Platform: platform,
			}
			vppAppTeam := fleet.VPPAppTeam{
				VPPAppID: vppAppID,
			}
			app := &fleet.VPPApp{
				VPPAppTeam:       vppAppTeam,
				BundleIdentifier: m.BundleID,
				IconURL:          m.ArtworkURL,
				Name:             m.TrackName,
				LatestVersion:    m.Version,
			}

			if appFleet, ok := assignedApps[vppAppID]; ok {
				// Then this is already assigned, so filter it out.
				app.SelfService = appFleet.SelfService
				appsToUpdate = append(appsToUpdate, app)
				continue
			}

			apps = append(apps, app)
		}
	}

	if len(appsToUpdate) > 0 {
		if err := svc.ds.BatchInsertVPPApps(ctx, appsToUpdate); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "updating existing VPP apps")
		}
	}

	// Sort apps by name and by platform
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Name != apps[j].Name {
			return apps[i].Name < apps[j].Name
		}
		return apps[i].Platform < apps[j].Platform
	})

	return apps, nil
}

func getPlatformsFromSupportedDevices(supportedDevices []string) map[fleet.AppleDevicePlatform]struct{} {
	platforms := make(map[fleet.AppleDevicePlatform]struct{}, 1)
	if len(supportedDevices) == 0 {
		platforms[fleet.MacOSPlatform] = struct{}{}
		return platforms
	}
	for _, device := range supportedDevices {
		// It is rare that a single app supports all platforms, but it is possible.
		switch {
		case strings.HasPrefix(device, "iPhone"):
			platforms[fleet.IOSPlatform] = struct{}{}
		case strings.HasPrefix(device, "iPad"):
			platforms[fleet.IPadOSPlatform] = struct{}{}
		case strings.HasPrefix(device, "Mac"):
			platforms[fleet.MacOSPlatform] = struct{}{}
		}
	}
	return platforms
}

func (svc *Service) AddAppStoreApp(ctx context.Context, teamID *uint, appID fleet.VPPAppTeam) error {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	// Validate platform
	if appID.Platform == "" {
		appID.Platform = fleet.MacOSPlatform
	}
	if appID.Platform != fleet.IOSPlatform && appID.Platform != fleet.IPadOSPlatform && appID.Platform != fleet.MacOSPlatform {
		return fleet.NewInvalidArgumentError("platform",
			fmt.Sprintf("platform must be one of '%s', '%s', or '%s", fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform))
	}

	var teamName string
	if teamID != nil && *teamID != 0 {
		tm, err := svc.ds.Team(ctx, *teamID)
		if fleet.IsNotFound(err) {
			return fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if team exists")
		}

		teamName = tm.Name
	}

	if appID.SelfService && appID.Platform != fleet.MacOSPlatform {
		return fleet.NewUserMessageError(errors.New("Currently, self-service only supports macOS"), http.StatusBadRequest)
	}

	vppToken, err := svc.getVPPToken(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(vppToken, &vpp.AssetFilter{AdamID: appID.AdamID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP asset")
	}

	if len(assets) == 0 {
		return ctxerr.New(ctx,
			fmt.Sprintf("Error: Couldn't add software. %s isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.",
				appID.AdamID))
	}

	asset := assets[0]

	assetMetadata, err := itunes.GetAssetMetadata([]string{asset.AdamID}, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	assetMD := assetMetadata[asset.AdamID]

	platforms := getPlatformsFromSupportedDevices(assetMD.SupportedDevices)
	if _, ok := platforms[appID.Platform]; !ok {
		return fleet.NewInvalidArgumentError("app_store_id", fmt.Sprintf("%s isn't available for %s", assetMD.TrackName, appID.Platform))
	}

	if appID.Platform == fleet.MacOSPlatform {
		// Check if we've already added an installer for this app
		exists, err := svc.ds.UploadedSoftwareExists(ctx, assetMD.BundleID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking existence of VPP app installer")
		}

		if exists {
			return ctxerr.New(ctx,
				fmt.Sprintf("Error: Couldn't add software. %s already has software available for install on the %s team.",
					assetMD.TrackName, teamName))
		}
	}

	app := &fleet.VPPApp{
		VPPAppTeam:       appID,
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
		Platform:      app.Platform,
		TeamName:      &teamName,
		SoftwareTitle: app.Name,
		TeamID:        teamID,
		SelfService:   app.SelfService,
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for add app store app")
	}

	return nil
}

func getVPPAppsMetadata(ctx context.Context, ids []fleet.VPPAppTeam) ([]*fleet.VPPApp, error) {
	var apps []*fleet.VPPApp

	// Map of adamID to platform, then to whether it's available as self-service.
	var adamIDMap = make(map[string]map[fleet.AppleDevicePlatform]bool)
	for _, id := range ids {
		if _, ok := adamIDMap[id.AdamID]; !ok {
			adamIDMap[id.AdamID] = make(map[fleet.AppleDevicePlatform]bool, 1)
			adamIDMap[id.AdamID][id.Platform] = id.SelfService
		} else {
			adamIDMap[id.AdamID][id.Platform] = id.SelfService
		}
	}

	var adamIDs []string
	for adamID := range adamIDMap {
		adamIDs = append(adamIDs, adamID)
	}
	assetMetatada, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	for adamID, metadata := range assetMetatada {
		platforms := getPlatformsFromSupportedDevices(metadata.SupportedDevices)
		for platform := range platforms {
			if selfService, ok := adamIDMap[adamID][platform]; ok {
				app := &fleet.VPPApp{
					VPPAppTeam: fleet.VPPAppTeam{
						VPPAppID: fleet.VPPAppID{
							AdamID:   adamID,
							Platform: platform,
						},
						SelfService: selfService,
					},
					BundleIdentifier: metadata.BundleID,
					IconURL:          metadata.ArtworkURL,
					Name:             metadata.TrackName,
					LatestVersion:    metadata.Version,
				}
				apps = append(apps, app)
			} else {
				continue
			}
		}
	}

	return apps, nil
}
