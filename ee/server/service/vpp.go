package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log/level"
)

// Used for overriding the env var value in testing
var testSetEmptyPrivateKey bool

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

var isAdamID = regexp.MustCompile(`^[0-9]+$`)

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []fleet.VPPBatchPayload, dryRun bool) ([]fleet.VPPAppResponse, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	var teamID *uint
	if teamName != "" {
		tm, err := svc.ds.TeamByName(ctx, teamName)
		if err != nil {
			// If this is a dry run, the team may not have been created yet
			if dryRun && fleet.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		teamID = &tm.ID
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating authorization")
	}

	// Adding VPP apps will add them to all available platforms per decision:
	// https://github.com/fleetdm/fleet/issues/19447#issuecomment-2256598681
	// The code is already here to support individual platforms, so we can easily enable it later.

	payloadsWithPlatform := make([]fleet.VPPBatchPayloadWithPlatform, 0, len(payloads))
	for _, payload := range payloads {
		if payload.Platform == "" && isAdamID.MatchString(payload.AppStoreID) {
			// add all possible Apple platforms, we'll remove the ones that this app doesn't support later
			payloadsWithPlatform = append(payloadsWithPlatform,
				fleet.VPPBatchPayloadWithPlatform{
					AppStoreID:          payload.AppStoreID,
					SelfService:         payload.SelfService,
					InstallDuringSetup:  payload.InstallDuringSetup,
					Platform:            fleet.MacOSPlatform,
					LabelsExcludeAny:    payload.LabelsExcludeAny,
					LabelsIncludeAny:    payload.LabelsIncludeAny,
					Categories:          payload.Categories,
					DisplayName:         payload.DisplayName,
					AutoUpdateEnabled:   payload.AutoUpdateEnabled,
					AutoUpdateStartTime: payload.AutoUpdateStartTime,
					AutoUpdateEndTime:   payload.AutoUpdateEndTime,
				},
				fleet.VPPBatchPayloadWithPlatform{
					AppStoreID:          payload.AppStoreID,
					SelfService:         payload.SelfService,
					InstallDuringSetup:  payload.InstallDuringSetup,
					Platform:            fleet.IOSPlatform,
					LabelsExcludeAny:    payload.LabelsExcludeAny,
					LabelsIncludeAny:    payload.LabelsIncludeAny,
					Categories:          payload.Categories,
					DisplayName:         payload.DisplayName,
					AutoUpdateEnabled:   payload.AutoUpdateEnabled,
					AutoUpdateStartTime: payload.AutoUpdateStartTime,
					AutoUpdateEndTime:   payload.AutoUpdateEndTime,
				},
				fleet.VPPBatchPayloadWithPlatform{
					AppStoreID:          payload.AppStoreID,
					SelfService:         payload.SelfService,
					InstallDuringSetup:  payload.InstallDuringSetup,
					Platform:            fleet.IPadOSPlatform,
					LabelsExcludeAny:    payload.LabelsExcludeAny,
					LabelsIncludeAny:    payload.LabelsIncludeAny,
					Categories:          payload.Categories,
					DisplayName:         payload.DisplayName,
					AutoUpdateEnabled:   payload.AutoUpdateEnabled,
					AutoUpdateStartTime: payload.AutoUpdateStartTime,
					AutoUpdateEndTime:   payload.AutoUpdateEndTime,
				},
			)
		}

		payloadsWithPlatform = append(payloadsWithPlatform, fleet.VPPBatchPayloadWithPlatform{
			AppStoreID:          payload.AppStoreID,
			SelfService:         payload.SelfService,
			InstallDuringSetup:  payload.InstallDuringSetup,
			Platform:            payload.Platform,
			LabelsExcludeAny:    payload.LabelsExcludeAny,
			LabelsIncludeAny:    payload.LabelsIncludeAny,
			Categories:          payload.Categories,
			DisplayName:         payload.DisplayName,
			Configuration:       payload.Configuration,
			AutoUpdateEnabled:   payload.AutoUpdateEnabled,
			AutoUpdateStartTime: payload.AutoUpdateStartTime,
			AutoUpdateEndTime:   payload.AutoUpdateEndTime,
		})

	}

	var incomingAppleApps, incomingAndroidApps []fleet.VPPAppTeam
	var vppToken string
	// Don't check for token if we're only disassociating assets
	if len(payloads) > 0 {
		for _, payload := range payloadsWithPlatform {
			if payload.Platform == "" {
				payload.Platform = fleet.MacOSPlatform
			}
			if !payload.Platform.SupportsAppStoreApps() {
				return nil, fleet.NewInvalidArgumentError("app_store_apps.platform",
					fmt.Sprintf("platform must be one of '%s', '%s', '%s', or '%s'", fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform, fleet.AndroidPlatform))
			}

			// Block Fleet Agent apps from being added via GitOps
			if payload.Platform == fleet.AndroidPlatform && strings.HasPrefix(payload.AppStoreID, fleetAgentPackagePrefix) {
				return nil, fleet.NewInvalidArgumentError("app_store_id", "The Fleet agent cannot be added manually. "+
					"It is automatically managed by Fleet when Android MDM is enabled.")
			}

			var err error
			if payload.Platform.IsApplePlatform() && vppToken == "" {
				vppToken, err = svc.getVPPToken(ctx, teamID)
				if err != nil {
					return nil, fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "could not retrieve vpp token"), http.StatusUnprocessableEntity)
				}
			}

			validatedLabels, err := ValidateSoftwareLabels(ctx, svc, teamID, payload.LabelsIncludeAny, payload.LabelsExcludeAny)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "validating software labels for batch adding vpp app")
			}

			payload.Categories = server.RemoveDuplicatesFromSlice(payload.Categories)
			catIDs, err := svc.ds.GetSoftwareCategoryIDs(ctx, payload.Categories)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "getting software category ids")
			}

			if len(catIDs) != len(payload.Categories) {
				return nil, &fleet.BadRequestError{
					Message:     "some or all of the categories provided don't exist",
					InternalErr: fmt.Errorf("categories provided: %v", payload.Categories),
				}
			}

			appStoreApp := fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   payload.AppStoreID,
					Platform: payload.Platform,
				},
				SelfService:         payload.SelfService,
				InstallDuringSetup:  payload.InstallDuringSetup,
				ValidatedLabels:     validatedLabels,
				CategoryIDs:         catIDs,
				DisplayName:         ptr.String(payload.DisplayName),
				AutoUpdateEnabled:   payload.AutoUpdateEnabled,
				AutoUpdateStartTime: payload.AutoUpdateStartTime,
				AutoUpdateEndTime:   payload.AutoUpdateEndTime,
			}
			switch payload.Platform {
			case fleet.AndroidPlatform:
				appStoreApp.SelfService = true
				appStoreApp.Configuration = payload.Configuration
				incomingAndroidApps = append(incomingAndroidApps, appStoreApp)
			case fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform:
				incomingAppleApps = append(incomingAppleApps, appStoreApp)
			}

		}

		if len(incomingAppleApps) > 0 {
			if dryRun {
				// If we're doing a dry run, we stop here and return no error to avoid making any changes.
				// That way we validate if a VPP token is available even on dry runs keeping it consistent.
				return nil, nil
			}

			var missingAssets []string

			assets, err := vpp.GetAssets(ctx, vppToken, nil)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "unable to retrieve assets")
			}

			assetMap := map[string]struct{}{}
			for _, asset := range assets {
				assetMap[asset.AdamID] = struct{}{}
			}

			for _, vppAppID := range incomingAppleApps {
				if _, ok := assetMap[vppAppID.AdamID]; !ok {
					missingAssets = append(missingAssets, vppAppID.AdamID)
				}
			}

			if len(missingAssets) != 0 {
				reqErr := ctxerr.Errorf(ctx, "requested app not available on vpp account: %s", strings.Join(missingAssets, ","))
				return nil, fleet.NewUserMessageError(reqErr, http.StatusUnprocessableEntity)
			}
		}
	}

	if dryRun {
		// If we're doing a dry run, we stop here and return no error to avoid making any changes.
		// Another dry run check is inside the payload size > 0 statement.
		return nil, nil
	}

	allPlatformApps := slices.Concat(incomingAppleApps, incomingAndroidApps)

	var appStoreApps []*fleet.VPPApp

	if len(incomingAppleApps) > 0 {
		apps, err := getVPPAppsMetadata(ctx, incomingAppleApps, vppToken, svc.getVPPAuthenticator(ctx))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "refreshing VPP app metadata")
		}
		if len(apps) == 0 {
			return nil, fleet.NewInvalidArgumentError("app_store_apps",
				"no valid apps found matching the provided app store IDs and platforms")
		}

		appStoreApps = append(appStoreApps, apps...)
	}

	var enterprise *android.Enterprise
	if len(incomingAndroidApps) > 0 {
		var err error
		enterprise, err = svc.ds.GetEnterprise(ctx)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: "Android MDM is not enabled", InternalErr: err}
		}

		for _, a := range incomingAndroidApps {
			androidApp, err := svc.androidModule.EnterprisesApplications(ctx, enterprise.Name(), a.AdamID)
			if err != nil {
				if fleet.IsNotFound(err) {
					return nil, fleet.NewInvalidArgumentError("app_store_id", "Couldn't add software. The application ID isn't available in Play Store. Please find ID on the Play Store and try again.")
				}
				return nil, ctxerr.Wrap(ctx, err, "bulk add app store apps: check if android app exists")
			}

			appStoreApps = append(appStoreApps, &fleet.VPPApp{
				VPPAppTeam:       a,
				BundleIdentifier: a.AdamID,
				IconURL:          androidApp.IconUrl,
				Name:             androidApp.Title,
				TeamID:           teamID,
			})
		}
	}

	if len(appStoreApps) > 0 {
		if err := svc.ds.BatchInsertVPPApps(ctx, appStoreApps); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "inserting vpp app metadata")
		}
	}

	appStoreIDToTitleID := make(map[string]uint, len(appStoreApps))
	for _, a := range appStoreApps {
		// The string representation includes the adam ID AND the platform, so it's unique per software title.
		appStoreIDToTitleID[a.VPPAppID.String()] = a.TitleID
	}

	// Filter out the apps with invalid platforms
	if len(appStoreApps) != len(allPlatformApps) {
		allPlatformApps = make([]fleet.VPPAppTeam, 0, len(appStoreApps))
		for _, app := range appStoreApps {
			allPlatformApps = append(allPlatformApps, app.VPPAppTeam)
		}
	}

	setupExperienceChanged, err := svc.ds.SetTeamVPPApps(ctx, teamID, allPlatformApps, appStoreIDToTitleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fleet.NewUserMessageError(ctxerr.Wrap(ctx, err, "no vpp token to set team vpp assets"), http.StatusUnprocessableEntity)
		}
		return nil, ctxerr.Wrap(ctx, err, "set team vpp assets")
	}

	// Do cleanup here because this is API call 2 of 2 for setting software from GitOps
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	// Apply auto-update config for iOS/iPadOS VPP apps
	// First, get existing auto-update schedules to know which apps already have configs
	existingIosAppSchedules, err := svc.ds.ListSoftwareAutoUpdateSchedules(ctx, tmID, "ios_apps")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing existing auto-update schedules for ios apps")
	}
	existingIPadOsSchedules, err := svc.ds.ListSoftwareAutoUpdateSchedules(ctx, tmID, "ipados_apps")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing existing auto-update schedules for ipados apps")
	}
	// Combine schedules from both sources
	existingSchedules := slices.Concat(existingIosAppSchedules, existingIPadOsSchedules)
	existingSchedulesByTitleID := make(map[uint]bool, len(existingSchedules))
	for _, schedule := range existingSchedules {
		existingSchedulesByTitleID[schedule.TitleID] = true
	}

	for _, app := range allPlatformApps {
		if app.Platform != fleet.IOSPlatform && app.Platform != fleet.IPadOSPlatform {
			continue
		}
		titleID, ok := appStoreIDToTitleID[app.VPPAppID.String()]
		if !ok {
			level.Error(svc.logger).Log("msg", "software title missing for vpp app", "vpp_app_id", app.VPPAppID.String())
			continue
		}

		hasAutoUpdateSettings := app.AutoUpdateEnabled != nil || app.AutoUpdateStartTime != nil || app.AutoUpdateEndTime != nil
		hasExistingSchedule := existingSchedulesByTitleID[titleID]

		// Only update if: app has auto update settings OR app has an existing schedule to disable
		if !hasAutoUpdateSettings && !hasExistingSchedule {
			continue
		}

		cfg := fleet.SoftwareAutoUpdateConfig{
			AutoUpdateEnabled:   app.AutoUpdateEnabled,
			AutoUpdateStartTime: app.AutoUpdateStartTime,
			AutoUpdateEndTime:   app.AutoUpdateEndTime,
		}

		if app.AutoUpdateEnabled == nil {
			cfg.AutoUpdateEnabled = ptr.Bool(false)
		}

		// Validate auto-update window if enabled or if times are provided
		hasTimesSet := app.AutoUpdateStartTime != nil || app.AutoUpdateEndTime != nil
		if (app.AutoUpdateEnabled != nil && *app.AutoUpdateEnabled) || hasTimesSet {
			schedule := fleet.SoftwareAutoUpdateSchedule{SoftwareAutoUpdateConfig: cfg}
			if err := schedule.WindowIsValid(); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "invalid auto-update window for vpp app")
			}
		}
		if err := svc.ds.UpdateSoftwareTitleAutoUpdateConfig(ctx, titleID, tmID, cfg); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "updating auto-update config for vpp app")
		}
	}

	if err := svc.ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, tmID); err != nil {
		return nil, err // returned error already includes context that we could include here
	}

	if len(allPlatformApps) == 0 {
		return []fleet.VPPAppResponse{}, nil
	}

	addedApps, err := svc.ds.GetVPPApps(ctx, teamID)
	if err != nil {
		return nil, err
	}

	policiesToUpdate := map[string]string{}
	var appIDs []string
	for _, app := range addedApps {
		if app.Platform == fleet.AndroidPlatform {
			hostsInScope, err := svc.ds.GetIncludedHostUUIDMapForAppStoreApp(ctx, app.AppTeamID)
			if err != nil {
				return nil, err
			}

			maps.Copy(policiesToUpdate, hostsInScope)
			appIDs = append(appIDs, app.AppStoreID)
		}
	}

	if len(policiesToUpdate) > 0 && enterprise != nil {
		for hostUUID, policyID := range policiesToUpdate {
			err := worker.QueueBulkSetAndroidAppsAvailableForHost(ctx, svc.ds, svc.logger, hostUUID, policyID, appIDs, enterprise.Name())
			if err != nil {
				return nil, ctxerr.WrapWithData(
					ctx,
					err,
					"batch associate app store apps: add apps to android MDM policy",
					map[string]any{
						"policy_id":       policyID,
						"host_uuid":       hostUUID,
						"application_ids": appIDs,
					},
				)
			}
		}
	}

	if setupExperienceChanged {
		err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedSetupExperienceSoftware{TeamID: ptr.ValOrZero(teamID), TeamName: teamName})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create edited setup experience activity")
		}
	}

	return addedApps, nil
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*fleet.VPPApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vppToken, err := svc.getVPPToken(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(ctx, vppToken, nil)
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

	metadata, err := apple_apps.GetMetadata(adamIDs, vppToken, svc.getVPPAuthenticator(ctx))
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
		m, ok := metadata[a.AdamID]
		if !ok {
			// Then this adam_id is not a VPP software entity, so skip it.
			continue
		}

		for _, app := range apple_apps.ToVPPApps(m) {
			if appFleet, ok := assignedApps[app.VPPAppID]; ok {
				// Then this is already assigned, so filter it out.
				app.SelfService = appFleet.SelfService
				appsToUpdate = append(appsToUpdate, &app)
				continue
			}

			apps = append(apps, &app)
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

var androidApplicationID = regexp.MustCompile(`^([A-Za-z]{1}[A-Za-z\d_]*\.)+[A-Za-z][A-Za-z\d_]*$`)

// fleetAgentPackagePrefix is the package prefix for Fleet Android agent.
// IT admins should not be able to add this app manually via the Software page as it is managed automatically by Fleet.
const fleetAgentPackagePrefix = "com.fleetdm.agent"

func (svc *Service) AddAppStoreApp(ctx context.Context, teamID *uint, appID fleet.VPPAppTeam) (uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return 0, err
	}
	if appID.AddAutoInstallPolicy {
		// Currently, same write permissions are applied on software and policies,
		// but leaving this here in case it changes in the future.
		if err := svc.authz.Authorize(ctx, &fleet.Policy{PolicyData: fleet.PolicyData{TeamID: teamID}}, fleet.ActionWrite); err != nil {
			return 0, err
		}
	}

	// Validate platform
	if appID.Platform == "" {
		appID.Platform = fleet.MacOSPlatform
	}

	if !appID.Platform.SupportsAppStoreApps() {
		return 0, fleet.NewInvalidArgumentError("platform",
			fmt.Sprintf("platform must be one of '%s', '%s', '%s', or '%s'", fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform, fleet.AndroidPlatform))
	}

	validatedLabels, err := ValidateSoftwareLabels(ctx, svc, teamID, appID.LabelsIncludeAny, appID.LabelsExcludeAny)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "validating software labels for adding vpp app")
	}

	var teamName string
	if teamID != nil && *teamID != 0 {
		tm, err := svc.ds.TeamLite(ctx, *teamID)
		if fleet.IsNotFound(err) {
			return 0, fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		} else if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "checking if team exists")
		}

		teamName = tm.Name
	}

	if appID.AddAutoInstallPolicy && appID.Platform != fleet.MacOSPlatform {
		return 0, fleet.NewUserMessageError(errors.New("Currently, automatic install is only supported on macOS, Windows, and Linux. Please add the app without automatic_install and manually install it on the Host details page."), http.StatusBadRequest)
	}

	isAndroidAppID := androidApplicationID.MatchString(appID.AdamID)

	var app *fleet.VPPApp
	var androidEnterpriseName string

	// Different flows based on platform
	switch appID.Platform {
	case fleet.AndroidPlatform:
		if !isAndroidAppID {
			return 0, fleet.NewInvalidArgumentError("app_store_id", "Application ID must be a valid Android application ID")
		}
		if strings.HasPrefix(appID.AdamID, fleetAgentPackagePrefix) {
			return 0, fleet.NewInvalidArgumentError("app_store_id", "The Fleet agent cannot be added manually. "+
				"It is automatically managed by Fleet when Android MDM is enabled.")
		}
		appID.SelfService = true
		appID.AddAutoInstallPolicy = false

		enterprise, err := svc.ds.GetEnterprise(ctx)
		if err != nil {
			return 0, &fleet.BadRequestError{Message: "Android MDM is not enabled", InternalErr: err}
		}
		androidEnterpriseName = enterprise.Name()

		androidApp, err := svc.androidModule.EnterprisesApplications(ctx, androidEnterpriseName, appID.AdamID)
		if err != nil {
			if fleet.IsNotFound(err) {
				return 0, fleet.NewInvalidArgumentError("app_store_id", "Couldn't add software. The application ID isn't available in Play Store. Please find ID on the Play Store and try again.")
			}
			return 0, ctxerr.Wrap(ctx, err, "add app store app: check if android app exists")
		}

		app = &fleet.VPPApp{
			VPPAppTeam:       appID,
			BundleIdentifier: appID.AdamID,
			IconURL:          androidApp.IconUrl,
			Name:             androidApp.Title,
			TeamID:           teamID,
		}

	default:
		if isAndroidAppID {
			return 0, fleet.NewInvalidArgumentError(
				"app_store_id",
				fmt.Sprintf(
					"Couldn't add software. %s isn't available in Apple Business Manager or Play Store. Please purchase a license in Apple Business Manager or find the app in Play Store and try again.",
					appID.AdamID,
				),
			)
		}

		vppToken, err := svc.getVPPToken(ctx, teamID)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "retrieving VPP token")
		}

		assets, err := vpp.GetAssets(ctx, vppToken, &vpp.AssetFilter{AdamID: appID.AdamID})
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "retrieving VPP asset")
		}

		if len(assets) == 0 {
			return 0, fleet.NewInvalidArgumentError("app_store_id",
				fmt.Sprintf("Error: Couldn't add software. %s isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.", appID.AdamID))
		}

		asset := assets[0]

		assetMetadata, err := apple_apps.GetMetadata([]string{asset.AdamID}, vppToken, svc.getVPPAuthenticator(ctx))
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
		}

		assetMD := assetMetadata[asset.AdamID]

		// Configuration is an Android only feature
		appID.Configuration = nil

		platforms := apple_apps.ToVPPApps(assetMD)
		appFromApple, ok := platforms[appID.Platform]
		if !ok {
			return 0, fleet.NewInvalidArgumentError("app_store_id", fmt.Sprintf("%s isn't available for %s", assetMD.Attributes.Name, appID.Platform))
		}

		if appID.Platform == fleet.MacOSPlatform {
			// Check if we've already added an installer for this app
			exists, err := svc.ds.UploadedSoftwareExists(ctx, appFromApple.BundleIdentifier, teamID)
			if err != nil {
				return 0, ctxerr.Wrap(ctx, err, "checking existence of VPP app installer")
			}

			if exists {
				return 0, ctxerr.Wrap(ctx, fleet.ConflictError{
					Message: fmt.Sprintf(fleet.CantAddSoftwareConflictMessage,
						assetMD.Attributes.Name, teamName),
				}, "vpp app conflicts with existing software installer")
			}
		}

		appID.ValidatedLabels = validatedLabels

		appID.Categories = server.RemoveDuplicatesFromSlice(appID.Categories)
		catIDs, err := svc.ds.GetSoftwareCategoryIDs(ctx, appID.Categories)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "getting software category ids")
		}

		if len(catIDs) != len(appID.Categories) {
			return 0, &fleet.BadRequestError{
				Message:     "some or all of the categories provided don't exist",
				InternalErr: fmt.Errorf("categories provided: %v", appID.Categories),
			}
		}
		appID.CategoryIDs = catIDs
		app = &appFromApple
		app.VPPAppTeam = appID
	}

	var androidConfigChanged bool
	// note that if appID.Configuration is nil, InsertVPPAppWithTeam will ignore it (it will not
	// update or remove it), so here we ignore it too if it is nil.
	if appID.Configuration != nil && appID.Platform == fleet.AndroidPlatform {
		changed, err := svc.ds.HasAndroidAppConfigurationChanged(ctx, appID.AdamID, ptr.ValOrZero(teamID), appID.Configuration)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "checking android app configuration change")
		}
		androidConfigChanged = changed
	}

	addedApp, err := svc.ds.InsertVPPAppWithTeam(ctx, app, teamID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "writing VPP app to db")
	}
	if appID.Platform == fleet.AndroidPlatform {
		err := worker.QueueMakeAndroidAppAvailableJob(ctx, svc.ds, svc.logger, appID.AdamID, addedApp.AppTeamID, androidEnterpriseName, androidConfigChanged)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "enqueuing job to make android app available")
		}
	}

	actLabelsIncl, actLabelsExcl := activitySoftwareLabelsFromValidatedLabels(addedApp.ValidatedLabels)

	act := fleet.ActivityAddedAppStoreApp{
		AppStoreID:       app.AdamID,
		Platform:         app.Platform,
		TeamName:         &teamName,
		SoftwareTitle:    app.Name,
		SoftwareTitleId:  addedApp.TitleID,
		TeamID:           teamID,
		SelfService:      app.SelfService,
		LabelsIncludeAny: actLabelsIncl,
		LabelsExcludeAny: actLabelsExcl,
		Configuration:    app.Configuration,
	}

	if err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create activity for add app store app")
	}

	if appID.AddAutoInstallPolicy && app.AddedAutomaticInstallPolicy != nil {
		policyAct := fleet.ActivityTypeCreatedPolicy{
			ID:   app.AddedAutomaticInstallPolicy.ID,
			Name: app.AddedAutomaticInstallPolicy.Name,
		}

		if err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), policyAct); err != nil {
			level.Warn(svc.logger).Log("msg", "failed to create activity for create automatic install policy for app store app", "err", err)
		}

	}

	return addedApp.TitleID, nil
}

func (svc *Service) getVPPAuthenticator(ctx context.Context) apple_apps.Authenticator {
	return apple_apps.GetAuthenticator(ctx, svc.ds, svc.config.License.Key)
}

func getVPPAppsMetadata(ctx context.Context, ids []fleet.VPPAppTeam, vppToken string, vppAuthenticator apple_apps.Authenticator) ([]*fleet.VPPApp, error) {
	var apps []*fleet.VPPApp

	// Map of adamID to platform, then to whether it's available as self-service
	// and installed during setup.
	adamIDMap := make(map[string]map[fleet.InstallableDevicePlatform]fleet.VPPAppTeam)
	for _, id := range ids {
		if _, ok := adamIDMap[id.AdamID]; !ok {
			adamIDMap[id.AdamID] = make(map[fleet.InstallableDevicePlatform]fleet.VPPAppTeam, 1)
			adamIDMap[id.AdamID][id.Platform] = fleet.VPPAppTeam{
				SelfService:         id.SelfService,
				InstallDuringSetup:  id.InstallDuringSetup,
				ValidatedLabels:     id.ValidatedLabels,
				AppTeamID:           id.AppTeamID,
				Categories:          id.Categories,
				CategoryIDs:         id.CategoryIDs,
				DisplayName:         id.DisplayName,
				AutoUpdateEnabled:   id.AutoUpdateEnabled,
				AutoUpdateStartTime: id.AutoUpdateStartTime,
				AutoUpdateEndTime:   id.AutoUpdateEndTime,
			}
		} else {
			adamIDMap[id.AdamID][id.Platform] = fleet.VPPAppTeam{
				SelfService:         id.SelfService,
				InstallDuringSetup:  id.InstallDuringSetup,
				ValidatedLabels:     id.ValidatedLabels,
				AppTeamID:           id.AppTeamID,
				Categories:          id.Categories,
				CategoryIDs:         id.CategoryIDs,
				DisplayName:         id.DisplayName,
				AutoUpdateEnabled:   id.AutoUpdateEnabled,
				AutoUpdateStartTime: id.AutoUpdateStartTime,
				AutoUpdateEndTime:   id.AutoUpdateEndTime,
			}
		}
	}

	var adamIDs []string
	for adamID := range adamIDMap {
		adamIDs = append(adamIDs, adamID)
	}
	assetMetadata, err := apple_apps.GetMetadata(adamIDs, vppToken, vppAuthenticator)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	for adamID, metadata := range assetMetadata {
		platforms := apple_apps.ToVPPApps(metadata)
		for platform, retrievedApp := range platforms {
			if props, ok := adamIDMap[adamID][platform]; ok {
				app := &fleet.VPPApp{
					VPPAppTeam: fleet.VPPAppTeam{
						VPPAppID: fleet.VPPAppID{
							AdamID:   adamID,
							Platform: platform,
						},
						SelfService:         props.SelfService,
						InstallDuringSetup:  props.InstallDuringSetup,
						ValidatedLabels:     props.ValidatedLabels,
						AppTeamID:           props.AppTeamID,
						Categories:          props.Categories,
						CategoryIDs:         props.CategoryIDs,
						DisplayName:         props.DisplayName,
						AutoUpdateEnabled:   props.AutoUpdateEnabled,
						AutoUpdateStartTime: props.AutoUpdateStartTime,
						AutoUpdateEndTime:   props.AutoUpdateEndTime,
					},
					BundleIdentifier: retrievedApp.BundleIdentifier,
					IconURL:          retrievedApp.IconURL,
					Name:             retrievedApp.Name,
					LatestVersion:    retrievedApp.LatestVersion,
				}
				apps = append(apps, app)
			} else {
				continue
			}
		}
	}

	return apps, nil
}

func (svc *Service) UpdateAppStoreApp(ctx context.Context, titleID uint, teamID *uint, payload fleet.AppStoreAppUpdatePayload) (*fleet.VPPAppStoreApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// If there's an auto-update config, validate it.
	// Note that applying this config is done in a separate service method.
	schedule := fleet.SoftwareAutoUpdateSchedule{
		SoftwareAutoUpdateConfig: fleet.SoftwareAutoUpdateConfig{
			AutoUpdateEnabled:   payload.AutoUpdateEnabled,
			AutoUpdateStartTime: payload.AutoUpdateStartTime,
			AutoUpdateEndTime:   payload.AutoUpdateEndTime,
		},
	}

	if payload.AutoUpdateEnabled != nil && *payload.AutoUpdateEnabled {
		if err := schedule.WindowIsValid(); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: validating auto-update schedule")
		}
	}

	var teamName string
	if teamID != nil && *teamID != 0 {
		tm, err := svc.ds.TeamLite(ctx, *teamID)
		if fleet.IsNotFound(err) {
			return nil, fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		} else if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: checking if team exists")
		}

		teamName = tm.Name
	}

	var validatedLabels *fleet.LabelIdentsWithScope
	if payload.LabelsExcludeAny != nil || payload.LabelsIncludeAny != nil {
		var err error
		validatedLabels, err = ValidateSoftwareLabels(ctx, svc, teamID, payload.LabelsIncludeAny, payload.LabelsExcludeAny)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: validating software labels")
		}
	}

	meta, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: getting vpp app metadata")
	}

	if payload.DisplayName != nil && *payload.DisplayName != meta.DisplayName {
		trimmed := strings.TrimSpace(*payload.DisplayName)
		if trimmed == "" && *payload.DisplayName != "" {
			return nil, fleet.NewInvalidArgumentError("display_name", "Cannot have a display name that is all whitespace.")
		}

		*payload.DisplayName = trimmed
	}

	selfServiceVal := meta.SelfService
	if payload.SelfService != nil && meta.Platform != fleet.AndroidPlatform {
		selfServiceVal = *payload.SelfService
	}
	if payload.Configuration != nil && meta.Platform != fleet.AndroidPlatform {
		payload.Configuration = nil
	}

	appToWrite := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID: meta.AdamID, Platform: meta.Platform,
			},
			SelfService:     selfServiceVal,
			ValidatedLabels: validatedLabels,
			DisplayName:     payload.DisplayName,
			Configuration:   payload.Configuration,
		},
		TeamID:           teamID,
		TitleID:          titleID,
		BundleIdentifier: meta.BundleIdentifier,
		Name:             meta.Name,
		LatestVersion:    meta.LatestVersion,
	}
	if meta.IconURL != nil {
		appToWrite.IconURL = *meta.IconURL
	}

	if payload.Categories != nil {
		payload.Categories = server.RemoveDuplicatesFromSlice(payload.Categories)
		catIDs, err := svc.ds.GetSoftwareCategoryIDs(ctx, payload.Categories)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting software category ids")
		}

		if len(catIDs) != len(payload.Categories) {
			return nil, &fleet.BadRequestError{
				Message:     "some or all of the categories provided don't exist",
				InternalErr: fmt.Errorf("categories provided: %v", payload.Categories),
			}
		}

		appToWrite.CategoryIDs = catIDs
	}

	// check if labels have changed
	var existingLabels fleet.LabelIdentsWithScope
	switch {
	case len(meta.LabelsExcludeAny) > 0:
		existingLabels.LabelScope = fleet.LabelScopeExcludeAny
		existingLabels.ByName = make(map[string]fleet.LabelIdent, len(meta.LabelsExcludeAny))
		for _, l := range meta.LabelsExcludeAny {
			existingLabels.ByName[l.LabelName] = fleet.LabelIdent{LabelName: l.LabelName, LabelID: l.LabelID}
		}

	case len(meta.LabelsIncludeAny) > 0:
		existingLabels.LabelScope = fleet.LabelScopeIncludeAny
		existingLabels.ByName = make(map[string]fleet.LabelIdent, len(meta.LabelsIncludeAny))
		for _, l := range meta.LabelsIncludeAny {
			existingLabels.ByName[l.LabelName] = fleet.LabelIdent{LabelName: l.LabelName, LabelID: l.LabelID}
		}
	}
	var labelsChanged bool
	if validatedLabels != nil {
		labelsChanged = !validatedLabels.Equal(&existingLabels)
	}

	// Get the hosts that are NOT in label scope currently (before the update happens)
	var hostsNotInScope map[uint]struct{}
	if labelsChanged {
		hostsNotInScope, err = svc.ds.GetExcludedHostIDMapForVPPApp(ctx, meta.VPPAppsTeamsID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: getting hosts not in scope for vpp app")
		}
	}

	var androidConfigChanged bool
	// note that if appID.Configuration is nil, InsertVPPAppWithTeam will ignore it (it will not
	// update or remove it), so here we ignore it too if it is nil.
	if payload.Configuration != nil && meta.Platform == fleet.AndroidPlatform {
		// check if configuration has changed
		androidConfigChanged, err = svc.ds.HasAndroidAppConfigurationChanged(ctx, meta.AdamID, ptr.ValOrZero(teamID), payload.Configuration)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: checking if android app configuration changed")
		}
	}

	// Update the app
	insertedApp, err := svc.ds.InsertVPPAppWithTeam(ctx, appToWrite, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: write app to db")
	}

	// if labelsChanged, new hosts may require having the app made available, and if config
	// changed, the app policy must be updated.
	if meta.Platform == fleet.AndroidPlatform && (labelsChanged || androidConfigChanged) {
		enterprise, err := svc.ds.GetEnterprise(ctx)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: "Android MDM is not enabled", InternalErr: err}
		}
		err = worker.QueueMakeAndroidAppAvailableJob(ctx, svc.ds, svc.logger, appToWrite.AdamID, insertedApp.AppTeamID, enterprise.Name(), androidConfigChanged)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "enqueuing job to make android app available")
		}
	}

	if labelsChanged {
		// Get the hosts that are now IN label scope (after the update)
		hostsInScope, err := svc.ds.GetIncludedHostIDMapForVPPApp(ctx, meta.VPPAppsTeamsID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: getting hosts in scope for vpp app")
		}

		var hostsToClear []uint
		for id := range hostsInScope {
			if _, ok := hostsNotInScope[id]; ok {
				// it was not in scope but now it is, so we should clear policy status
				hostsToClear = append(hostsToClear, id)
			}
		}

		// We clear the policy status here because otherwise the policy automation machinery
		// won't pick this up and the software won't install.
		if err := svc.ds.ClearVPPAppAutoInstallPolicyStatusForHosts(ctx, meta.VPPAppsTeamsID, hostsToClear); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to clear auto install policy status for host")
		}
	}

	actLabelsIncl, actLabelsExcl := activitySoftwareLabelsFromValidatedLabels(validatedLabels)

	displayNameVal := ptr.ValOrZero(payload.DisplayName)

	act := fleet.ActivityEditedAppStoreApp{
		TeamName:            &teamName,
		TeamID:              teamID,
		SelfService:         selfServiceVal,
		SoftwareTitleID:     titleID,
		SoftwareTitle:       meta.Name,
		AppStoreID:          meta.AdamID,
		Platform:            meta.Platform,
		LabelsIncludeAny:    actLabelsIncl,
		LabelsExcludeAny:    actLabelsExcl,
		SoftwareIconURL:     meta.IconURL,
		SoftwareDisplayName: displayNameVal,
		Configuration:       appToWrite.Configuration,
	}

	updatedAppMeta, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "UpdateAppStoreApp: getting updated app metadata")
	}

	if payload.AutoUpdateEnabled != nil {
		// Update AutoUpdateConfig separately
		err = svc.UpdateSoftwareTitleAutoUpdateConfig(ctx, titleID, teamID, fleet.SoftwareAutoUpdateConfig{
			AutoUpdateEnabled:   payload.AutoUpdateEnabled,
			AutoUpdateStartTime: payload.AutoUpdateStartTime,
			AutoUpdateEndTime:   payload.AutoUpdateEndTime,
		})
		if err != nil {
			return nil, err
		}
	}

	// Re-fetch the software title to get the updated auto-update config.
	updatedTitle, err := svc.SoftwareTitleByID(ctx, titleID, teamID)
	if err != nil {
		return nil, err
	}
	if updatedTitle.AutoUpdateEnabled != nil {
		act.AutoUpdateEnabled = updatedTitle.AutoUpdateEnabled
		if *updatedTitle.AutoUpdateEnabled {
			act.AutoUpdateStartTime = updatedTitle.AutoUpdateStartTime
			act.AutoUpdateEndTime = updatedTitle.AutoUpdateEndTime
		}
	}

	if err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return nil, err
	}

	return updatedAppMeta, nil
}

func (svc *Service) UploadVPPToken(ctx context.Context, token io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't add content token. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}

	if token == nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
	}

	tokenBytes, err := io.ReadAll(token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading VPP token")
	}

	locName, err := vpp.GetConfig(string(tokenBytes))
	if err != nil {
		var vppErr *vpp.ErrorResponse
		if errors.As(err, &vppErr) {
			// Per https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			if vppErr.ErrorNumber == 9622 {
				return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "validating VPP token with Apple")
	}

	data := fleet.VPPTokenData{
		Token:    string(tokenBytes),
		Location: locName,
	}

	tok, err := svc.ds.InsertVPPToken(ctx, &data)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "writing VPP token to db")
	}

	if err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEnabledVPP{
		Location: locName,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for upload VPP token")
	}

	return tok, nil
}

func (svc *Service) UpdateVPPToken(ctx context.Context, tokenID uint, token io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't add content token. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}

	if token == nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
	}

	tokenBytes, err := io.ReadAll(token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading VPP token")
	}

	locName, err := vpp.GetConfig(string(tokenBytes))
	if err != nil {
		var vppErr *vpp.ErrorResponse
		if errors.As(err, &vppErr) {
			// Per https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			if vppErr.ErrorNumber == 9622 {
				return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "validating VPP token with Apple")
	}

	data := fleet.VPPTokenData{
		Token:    string(tokenBytes),
		Location: locName,
	}

	tok, err := svc.ds.UpdateVPPToken(ctx, tokenID, &data)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating vpp token")
	}

	return tok, nil
}

func (svc *Service) UpdateVPPTokenTeams(ctx context.Context, tokenID uint, teamIDs []uint) (*fleet.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	tok, err := svc.ds.UpdateVPPTokenTeams(ctx, tokenID, teamIDs)
	if err != nil {
		var errTokConstraint fleet.ErrVPPTokenTeamConstraint
		if errors.As(err, &errTokConstraint) {
			return nil, ctxerr.Wrap(ctx, fleet.NewUserMessageError(errTokConstraint, http.StatusConflict))
		}
		return nil, ctxerr.Wrap(ctx, err, "updating vpp token team")
	}

	return tok, nil
}

func (svc *Service) GetVPPTokens(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListVPPTokens(ctx)
}

func (svc *Service) DeleteVPPToken(ctx context.Context, tokenID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return err
	}
	tok, err := svc.ds.GetVPPToken(ctx, tokenID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting vpp token")
	}
	if err := svc.activitiesModule.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDisabledVPP{
		Location: tok.Location,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for delete VPP token")
	}

	return svc.ds.DeleteVPPToken(ctx, tokenID)
}
