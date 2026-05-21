package service

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

func (svc *Service) GetOrbitSetupExperienceStatus(ctx context.Context, orbitNodeKey string, forceRelease bool, resetFailedSetupSteps bool) (*fleet.SetupExperienceStatusPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return nil, err
	}

	if fleet.IsLinux(host.Platform) || host.Platform == "windows" {
		// Windows and Linux setup experience only have software.
		status, err := svc.getHostSetupExperienceStatus(ctx, host)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host setup experience status")
		}
		return &fleet.SetupExperienceStatusPayload{
			Software: status.Software,
		}, nil
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	// get the status of the bootstrap package deployment
	bootstrapPkg, err := svc.ds.GetHostMDMMacOSSetup(ctx, host.ID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package status")
	}

	// NOTE: bootstrapPkg can be nil if there was none to install.
	var bootstrapPkgResult *fleet.SetupExperienceBootstrapPackageResult
	if bootstrapPkg != nil {
		bootstrapPkgResult = &fleet.SetupExperienceBootstrapPackageResult{
			Name:   bootstrapPkg.BootstrapPackageName,
			Status: bootstrapPkg.BootstrapPackageStatus,
		}
	}

	// get the status of the configuration profiles
	cfgProfs, err := svc.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get configuration profiles status")
	}
	var cfgProfResults []*fleet.SetupExperienceConfigurationProfileResult
	for _, prof := range cfgProfs {
		// NOTE: DDM profiles (declarations) are ignored because while a device is
		// awaiting to be released, it cannot process a DDM session (at least
		// that's what we noticed during testing).
		if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			continue
		}

		// NOTE: user-scoped profiles are ignored because they are not sent by Fleet
		// until after the device is released - there is no user-channel available
		// on the host until after the release, and after the user actually created
		// the user account.
		if prof.Scope == fleet.PayloadScopeUser {
			continue
		}

		status := fleet.MDMDeliveryPending
		if prof.Status != nil {
			status = *prof.Status
		}
		cfgProfResults = append(cfgProfResults, &fleet.SetupExperienceConfigurationProfileResult{
			ProfileUUID: prof.ProfileUUID,
			Name:        prof.Name,
			Status:      status,
		})
	}

	profilesMissingInstallation, err := svc.ds.ListMDMAppleProfilesToInstall(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing apple config profiles to install")
	}
	profilesMissingInstallation = fleet.FilterOutUserScopedProfiles(profilesMissingInstallation)
	if host.Platform != "darwin" {
		profilesMissingInstallation = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(profilesMissingInstallation)
	}
	if len(profilesMissingInstallation) > 0 {
		for _, prof := range profilesMissingInstallation {
			cfgProfResults = append(cfgProfResults, &fleet.SetupExperienceConfigurationProfileResult{
				ProfileUUID: prof.ProfileUUID,
				Name:        prof.ProfileName,
				Status:      fleet.MDMDeliveryPending, // Default to pending as it's not installed yet.
			})
		}
	}

	// AccountConfiguration covers the (optional) command to setup SSO.
	adminTeamFilter := fleet.TeamFilter{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	}
	acctCmds, _, _, err := svc.ds.ListMDMCommands(ctx, adminTeamFilter, &fleet.MDMCommandListOptions{
		// PerPage 1: only acctCmds[0] is read below.
		ListOptions: fleet.ListOptions{PerPage: 1},
		Filters: fleet.MDMCommandFilters{
			HostIdentifier: host.UUID,
			RequestType:    "AccountConfiguration",
		},
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list AccountConfiguration commands")
	}

	var acctCfgResult *fleet.SetupExperienceAccountConfigurationResult
	if len(acctCmds) > 0 {
		// there may be more than one if e.g. the worker job that sends them had to
		// retry, but they would all be processed anyway so we can only care about
		// the first one.
		acctCfgResult = &fleet.SetupExperienceAccountConfigurationResult{
			CommandUUID: acctCmds[0].CommandUUID,
			Status:      acctCmds[0].Status,
		}
	}

	// get status of software installs and script execution
	res, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, ptr.ValOrZero(host.TeamID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	// Check if "require all software" is configured for the host's team.
	requireAllSoftware, err := svc.IsAllSetupExperienceSoftwareRequired(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if all software is required")
	}

	hasFailedSoftwareInstall := false
	for _, r := range res {
		if r.IsForSoftware() && r.Status == fleet.SetupExperienceStatusFailure {
			hasFailedSoftwareInstall = true
			break
		}
	}
	// If we have a failed software install,
	// AND "require all software" is configured for the host's team,
	// AND the resetFailedSetupSteps flag is set,
	// then re-enqueue any cancelled setup experience steps.
	if hasFailedSoftwareInstall {
		if resetFailedSetupSteps {
			// If so, call the enqueue function with a flag to retain successful steps.
			if requireAllSoftware {
				svc.logger.InfoContext(ctx, "re-enqueueing cancelled setup experience steps after a previous software install failure", "host_uuid", host.UUID)
				_, err := svc.ds.ResetSetupExperienceItemsAfterFailure(ctx, host.Platform, host.PlatformLike, host.UUID, ptr.ValOrZero(host.TeamID))
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "re-enqueueing cancelled setup experience steps after a previous software install failure")
				}
				// Re-fetch the setup experience results after re-enqueuing.
				res, err = svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, ptr.ValOrZero(host.TeamID))
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
				}
			}
		}
	}

	if err = svc.recordCanceledSetupExperienceSoftwareActivities(ctx, host.ID, host.UUID, host.DisplayName(), res); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "recording cancelled setup experience installs")
	}

	payload := &fleet.SetupExperienceStatusPayload{
		BootstrapPackage:      bootstrapPkgResult,
		ConfigurationProfiles: cfgProfResults,
		AccountConfiguration:  acctCfgResult,
		Software:              make([]*fleet.SetupExperienceStatusResult, 0),
		OrgLogoURL:            fleet.AbsolutizeLogoURL(appCfg.OrgInfo.OrgLogoURLLightBackground, appCfg.ServerSettings.ServerURL),
		RequireAllSoftware:    requireAllSoftware,
	}
	for _, r := range res {
		if r.IsForScript() {
			payload.Script = r
		}

		if r.IsForSoftware() {
			payload.Software = append(payload.Software, r)
		}
	}

	// If we have failed software, and all software is required,
	// we can go ahead and return now.
	if hasFailedSoftwareInstall && requireAllSoftware {
		return payload, nil
	}

	if forceRelease || isDeviceReadyForRelease(payload) {
		manual, err := isDeviceReleasedManually(ctx, svc.ds, host)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "check if device is released manually")
		}
		if manual {
			return payload, nil
		}

		// otherwise the device is not released manually, proceed with automatic
		// release
		if forceRelease {
			svc.logger.WarnContext(ctx, "force-releasing device, DEP enrollment commands, profiles, software installs and script execution may not have all completed", "host_uuid", host.UUID)
		} else {
			svc.logger.InfoContext(ctx, "releasing device, all DEP enrollment commands, profiles, software installs and script execution have completed", "host_uuid", host.UUID)
		}

		// Host will be marked as no longer "awaiting configuration" in the command handler
		if err := svc.mdmAppleCommander.DeviceConfigured(ctx, host.UUID, uuid.NewString()); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to enqueue DeviceConfigured command")
		}
	}

	_, err = svc.SetupExperienceNextStep(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting next step for host setup experience")
	}

	return payload, nil
}

func (svc *Service) recordCanceledSetupExperienceSoftwareActivities(
	ctx context.Context,
	hostID uint,
	hostUUID string,
	hostDisplayName string,
	results []*fleet.SetupExperienceStatusResult,
) error {
	for _, r := range results {
		if r.Status != fleet.SetupExperienceStatusCancelled {
			continue
		}
		r.Status = fleet.SetupExperienceStatusFailure
		svc.logger.InfoContext(ctx, "emitting activity for canceled setup experience software", "host_uuid", hostUUID, "software_name", r.Name)
		err := svc.ds.UpdateSetupExperienceStatusResult(ctx, r)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marking canceled setup experience software install as failed")
		}
		if r.IsForSoftwarePackage() {
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeCanceledInstallSoftware{
				HostID:              hostID,
				HostDisplayName:     hostDisplayName,
				SoftwareTitle:       r.Name,
				SoftwareTitleID:     ptr.ValOrZero(r.SoftwareTitleID),
				FromSetupExperience: true,
			}); err != nil {
				return ctxerr.Wrap(ctx, err, "creating activity for canceled setup experience software install")
			}
		} else if r.IsForVPPApp() {
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeCanceledInstallAppStoreApp{
				HostID:              hostID,
				HostDisplayName:     hostDisplayName,
				SoftwareTitle:       r.Name,
				SoftwareTitleID:     ptr.ValOrZero(r.SoftwareTitleID),
				FromSetupExperience: true,
			}); err != nil {
				return ctxerr.Wrap(ctx, err, "creating activity for canceled setup experience VPP app install")
			}
		}
	}

	return nil
}

func isDeviceReleasedManually(ctx context.Context, ds fleet.Datastore, host *fleet.Host) (bool, error) {
	var manualRelease bool
	if host.TeamID == nil {
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get AppConfig to read apple_enable_release_device_manually")
		}
		manualRelease = ac.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
	} else {
		tm, err := ds.TeamLite(ctx, *host.TeamID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get Team to read apple_enable_release_device_manually")
		}
		manualRelease = tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
	}
	return manualRelease, nil
}

func isDeviceReadyForRelease(payload *fleet.SetupExperienceStatusPayload) bool {
	// default to "do release" and return false as soon as we find a reason not
	// to.

	if payload.BootstrapPackage != nil {
		if payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageFailed &&
			payload.BootstrapPackage.Status != fleet.MDMBootstrapPackageInstalled {
			// bootstrap package is still pending, not ready for release
			return false
		}
	}

	if payload.AccountConfiguration != nil {
		if payload.AccountConfiguration.Status != fleet.MDMAppleStatusAcknowledged &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusError &&
			payload.AccountConfiguration.Status != fleet.MDMAppleStatusCommandFormatError {
			// account configuration command is still pending, not ready for release
			return false
		}
	}

	for _, prof := range payload.ConfigurationProfiles {
		if prof.Status != fleet.MDMDeliveryFailed &&
			prof.Status != fleet.MDMDeliveryVerifying &&
			prof.Status != fleet.MDMDeliveryVerified {
			// profile is still pending, not ready for release
			return false
		}
	}

	for _, sw := range payload.Software {
		if sw.Status != fleet.SetupExperienceStatusFailure &&
			sw.Status != fleet.SetupExperienceStatusSuccess {
			// software is still pending, not ready for release
			return false
		}
	}

	if payload.Script != nil {
		if payload.Script.Status != fleet.SetupExperienceStatusFailure &&
			payload.Script.Status != fleet.SetupExperienceStatusSuccess {
			// script is still pending, not ready for release
			return false
		}
	}

	return true
}

func (svc *Service) SetupExperienceInit(ctx context.Context) (*fleet.SetupExperienceInitResult, error) {
	// This is an orbit endpoint, not a user-authenticated endpoint.
	svc.authz.SkipAuthorization(ctx)

	// NOTE: currently, Android does not go through the "init" setup experience flow as it
	// doesn't support any on-device UI (such as the screen showing setup progress) nor any
	// ordering of installs - all software to install is provided as part of the Android policy
	// when the host enrolls in Fleet.
	// See https://github.com/fleetdm/fleet/issues/33761#issuecomment-3548996114

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "internal error: missing host from request context")
	}

	// teamID for EnqueueSetupExperienceItems should be 0 for "No team" hosts.
	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}

	hostUUID, err := fleet.HostUUIDForSetupExperience(host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to get host's UUID for the setup experience")
	}

	enabled, err := svc.ds.EnqueueSetupExperienceItems(ctx, host.Platform, host.PlatformLike, hostUUID, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "check for software titles for setup experience")
	}

	return &fleet.SetupExperienceInitResult{
		Enabled: enabled,
	}, nil
}
