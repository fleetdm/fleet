package worker

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

// Name of the Apple MDM job as registered in the worker. Note that although it
// is a single job, it can process a number of different-but-related tasks,
// identified by the Task field in the job's payload.
const appleMDMJobName = "apple_mdm"

type AppleMDMTask string

// List of supported tasks.
const (
	AppleMDMPostDEPEnrollmentTask    AppleMDMTask = "post_dep_enrollment"
	AppleMDMPostManualEnrollmentTask AppleMDMTask = "post_manual_enrollment"
	// PostDEPReleaseDevice is not enqueued anymore for macOS but remains for
	// backward compatibility (processing existing jobs after a fleet upgrade)
	// and for ios/ipados. Macs are now released via the swift dialog UI of the
	// setup experience flow.
	AppleMDMPostDEPReleaseDeviceTask AppleMDMTask = "post_dep_release_device"
)

// AppleMDM is the job processor for the apple_mdm job.
type AppleMDM struct {
	Datastore             fleet.Datastore
	Log                   *slog.Logger
	Commander             *apple_mdm.MDMAppleCommander
	BootstrapPackageStore fleet.MDMBootstrapPackageStore
	VPPInstaller          fleet.AppleMDMVPPInstaller
	NewActivityFn         fleet.NewActivityFunc
}

// Name returns the name of the job.
func (a *AppleMDM) Name() string {
	return appleMDMJobName
}

// appleMDMArgs is the payload for the Apple MDM job.
type appleMDMArgs struct {
	Task     AppleMDMTask `json:"task"`
	HostUUID string       `json:"host_uuid"`
	TeamID   *uint        `json:"team_id,omitempty"`
	// EnrollReference is the UUID of the MDM IdP account used to enroll the
	// device. It is used to set the username and full name of the user
	// associated with the device.
	//
	// FIXME: Rename this to IdPAccountUUID or something similar.
	EnrollReference        string     `json:"enroll_reference,omitempty"`
	EnrollmentCommands     []string   `json:"enrollment_commands,omitempty"`
	Platform               string     `json:"platform,omitempty"`
	UseWorkerDeviceRelease bool       `json:"use_worker_device_release,omitempty"`
	ReleaseDeviceAttempt   int        `json:"release_device_attempt,omitempty"`    // number of attempts to release the device
	ReleaseDeviceStartedAt *time.Time `json:"release_device_started_at,omitempty"` // time when the release device task first started
	FromMDMMigration       bool       `json:"from_mdm_migration,omitempty"`        // indicates if the task is part of an MDM migration
}

// Run executes the apple_mdm job.
func (a *AppleMDM) Run(ctx context.Context, argsJSON json.RawMessage) error {
	appCfg, err := a.Datastore.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}
	if !appCfg.MDM.EnabledAndConfigured || a.Commander == nil {
		return nil
	}

	var args appleMDMArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case AppleMDMPostDEPEnrollmentTask:
		err := a.runPostDEPEnrollment(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple DEP enrollment task")

	case AppleMDMPostManualEnrollmentTask:
		err := a.runPostManualEnrollment(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple manual enrollment task")

	case AppleMDMPostDEPReleaseDeviceTask:
		err := a.runPostDEPReleaseDevice(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple DEP release device task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func isMacOS(platform string) bool {
	// For backwards compatibility, we assume empty platform in job arguments is macOS.
	return platform == "" ||
		platform == "darwin"
}

func (a *AppleMDM) runPostManualEnrollment(ctx context.Context, args appleMDMArgs) error {
	_, err := a.installProfilesForEnrollingHost(ctx, args.HostUUID)
	if err != nil {
		a.Log.ErrorContext(ctx, "error installing profiles for enrolling host", "host_uuid", args.HostUUID, "err", err)
		// We do not return here, as we want to continue with the rest of the logic, and then the reconciler will just pick up the remaining work.
		// We do this since this is a speed optimization and not critical to complete enrollment itself.
	}

	if isMacOS(args.Platform) {
		if _, err := a.installFleetd(ctx, args.HostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
		}
	} else {
		// We shouldn't have any setup experience steps if we're not on a premium license,
		// but best to check anyway plus it saves some db queries.
		if license.IsPremium(ctx) {
			_, err := a.installSetupExperienceVPPAppsOnIosIpadOS(ctx, args.HostUUID, ptr.ValOrZero(args.TeamID))
			if err != nil {
				return ctxerr.Wrap(ctx, err, "installing setup experience VPP apps on iOS/iPadOS")
			}
		}
	}

	return nil
}

func (a *AppleMDM) runPostDEPEnrollment(ctx context.Context, args appleMDMArgs) error {
	var (
		awaitCmdUUIDs []string
		appCfg        *fleet.AppConfig
		team          *fleet.Team
		err           error
	)

	if isMacOS(args.Platform) {
		var manualAgentInstall bool
		if args.TeamID == nil {
			if appCfg, err = a.getAppConfig(ctx, appCfg); err != nil {
				return err
			}
			manualAgentInstall = appCfg.MDM.MacOSSetup.ManualAgentInstall.Value
		} else {
			if team, err = a.getTeamConfig(ctx, team, *args.TeamID); err != nil {
				return err
			}
			manualAgentInstall = team.Config.MDM.MacOSSetup.ManualAgentInstall.Value
		}

		if !manualAgentInstall {
			fleetdCmdUUID, err := a.installFleetd(ctx, args.HostUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
			}
			awaitCmdUUIDs = append(awaitCmdUUIDs, fleetdCmdUUID)
		}

		allowBootstrapDuringMigration := false
		allowBootstrapDuringMigrationEV := os.Getenv("FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION")
		if allowBootstrapDuringMigrationEV == "1" || strings.EqualFold(allowBootstrapDuringMigrationEV, "true") {
			allowBootstrapDuringMigration = true
		}

		if args.FromMDMMigration && !allowBootstrapDuringMigration {
			a.Log.InfoContext(ctx, "skipping bootstrap package installation during MDM migration", "host_uuid", args.HostUUID)
			err = a.Datastore.RecordSkippedHostBootstrapPackage(ctx, args.HostUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "recording skipped bootstrap package")
			}
		} else {
			bootstrapCmdUUID, err := a.installBootstrapPackage(ctx, args.HostUUID, args.TeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
			}
			if bootstrapCmdUUID != "" {
				awaitCmdUUIDs = append(awaitCmdUUIDs, bootstrapCmdUUID)
			}
		}
	} else {
		commandUUIDs, err := a.installSetupExperienceVPPAppsOnIosIpadOS(ctx, args.HostUUID, ptr.ValOrZero(args.TeamID))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "installing setup experience VPP apps on iOS/iPadOS")
		}
		awaitCmdUUIDs = append(awaitCmdUUIDs, commandUUIDs...)
	}

	cmdUUIDs, err := a.installProfilesForEnrollingHost(ctx, args.HostUUID)
	if err != nil {
		a.Log.ErrorContext(ctx, "error installing profiles for enrolling host", "host_uuid", args.HostUUID, "err", err)
		// We do not return here, as we want to continue with the rest of the logic, and then the reconciler will just pick up the remaining work.
		// We do this since this is a speed optimization and not critical to complete enrollment itself, as we have other backing logic.
		cmdUUIDs = []string{}
	}

	awaitCmdUUIDs = append(awaitCmdUUIDs, cmdUUIDs...)

	var ssoEnabled, managedAdminAccountEnabled, lockPrimaryAccountInfo bool
	var ssoAccount *fleet.MDMIdPAccount
	var adminAccount *apple_mdm.AdminAccountConfig

	if ref := args.EnrollReference; ref != "" {
		a.Log.InfoContext(ctx, "got an enroll_reference", "host_uuid", args.HostUUID, "ref", ref)
		if appCfg, err = a.getAppConfig(ctx, appCfg); err != nil {
			return err
		}

		ssoAccount, err = a.Datastore.GetMDMIdPAccountByUUID(ctx, ref)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "getting idp account details for enroll reference %s", ref)
		}

		ssoEnabled = appCfg.MDM.MacOSSetup.EnableEndUserAuthentication
		lockPrimaryAccountInfo = appCfg.MDM.MacOSSetup.LockEndUserInfo.Value
		if args.TeamID != nil {
			if team, err = a.getTeamConfig(ctx, team, *args.TeamID); err != nil {
				return err
			}
			ssoEnabled = team.Config.MDM.MacOSSetup.EnableEndUserAuthentication
			lockPrimaryAccountInfo = team.Config.MDM.MacOSSetup.LockEndUserInfo.Value
		}
	}

	if isMacOS(args.Platform) && license.IsPremium(ctx) {
		if args.TeamID == nil {
			if appCfg, err = a.getAppConfig(ctx, appCfg); err != nil {
				return err
			}
			managedAdminAccountEnabled = appCfg.MDM.MacOSSetup.EnableManagedLocalAccount.Value
		} else {
			if team, err = a.getTeamConfig(ctx, team, *args.TeamID); err != nil {
				return err
			}
			managedAdminAccountEnabled = team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value
		}
	}

	const fleetAdminFullName = "Fleet Admin"

	// Only send AccountConfiguration for macOS devices.
	if isMacOS(args.Platform) && (ssoEnabled || managedAdminAccountEnabled) {
		var password string
		cmdUUID := uuid.New().String()
		if managedAdminAccountEnabled {
			password = apple_mdm.GenerateManagedAccountPassword()
			passwordHash, err := apple_mdm.GenerateSaltedSHA512PBKDF2Hash(password)
			if err != nil {
				return err
			}
			adminAccount = &apple_mdm.AdminAccountConfig{
				ShortName:    fleet.ManagedLocalAccountUsername,
				FullName:     fleetAdminFullName,
				PasswordHash: passwordHash,
				Hidden:       true,
			}
			// Save the password before sending the command so the plaintext is
			// escrowed even if the command enqueue succeeds but a later step fails.
			if err := a.Datastore.SaveHostManagedLocalAccount(ctx, args.HostUUID, password, cmdUUID); err != nil {
				return err
			}
		}

		// Only include the SSO account in the payload if SSO is actually enabled.
		// ssoAccount may be non-nil (fetched from enroll reference) even when SSO is disabled.
		var ssoAccountForPayload *fleet.MDMIdPAccount
		if ssoEnabled {
			ssoAccountForPayload = ssoAccount
		}
		if err := a.sendManagedAccounts(ctx, &args, ssoAccountForPayload, adminAccount, lockPrimaryAccountInfo, cmdUUID); err != nil {
			return err
		}
		awaitCmdUUIDs = append(awaitCmdUUIDs, cmdUUID)
	}

	// proceed to release the device if it is not a macos, as those are released
	// via the setup experience flow, or if we were told to use the worker based
	// release.
	if !isMacOS(args.Platform) || args.UseWorkerDeviceRelease {
		var manualRelease bool
		if args.TeamID == nil {
			if appCfg, err = a.getAppConfig(ctx, appCfg); err != nil {
				return err
			}
			manualRelease = appCfg.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
		} else {
			if team, err = a.getTeamConfig(ctx, team, *args.TeamID); err != nil {
				return err
			}
			manualRelease = team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
		}

		if !manualRelease {
			// send all command uuids for the commands sent here during post-DEP
			// enrollment and enqueue a job to look for the status of those commands to
			// be final and same for MDM profiles of that host; it means the DEP
			// enrollment process is done and the device can be released.
			if err := QueueAppleMDMJob(ctx, a.Datastore, a.Log, AppleMDMPostDEPReleaseDeviceTask,
				args.HostUUID, args.Platform, args.TeamID, args.EnrollReference, false, args.FromMDMMigration, awaitCmdUUIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "queue Apple Post-DEP release device job")
			}
		}
	}

	return nil
}

// getTeamConfig gets team config from DB if not provided.
func (a *AppleMDM) getTeamConfig(ctx context.Context, team *fleet.Team, teamID uint) (*fleet.Team, error) {
	if team == nil { // TODO see if we can swap this (plus callers) to use TeamLite
		var err error
		team, err = a.Datastore.TeamWithExtras(ctx, teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetch team to send AccountConfiguration")
		}
	}
	return team, nil
}

// getAppConfig gets app config from DB if not provided.
func (a *AppleMDM) getAppConfig(ctx context.Context, appConfig *fleet.AppConfig) (*fleet.AppConfig, error) {
	if appConfig == nil {
		var err error
		appConfig, err = a.Datastore.AppConfig(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting app config")
		}
	}
	return appConfig, nil
}

func (a *AppleMDM) getIdPDisplayName(ctx context.Context, acct *fleet.MDMIdPAccount, args appleMDMArgs) (string, error) {
	if acct.Fullname != "" && !strings.Contains(acct.Fullname, "@") {
		return acct.Fullname, nil
	}

	// If full name is empty or appears to be an email, see if it exists via SCIM integration
	scimUser, err := a.Datastore.ScimUserByUserNameOrEmail(ctx, acct.Username, acct.Email)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		return "", ctxerr.Wrap(ctx, err, "getting scim user details for enroll reference %s and host_uuid %s", acct.UUID, args.HostUUID)
	case scimUser == nil:
		return acct.Fullname, nil
	}
	if scimUser.DisplayName() == "" {
		return acct.Fullname, nil
	}
	return scimUser.DisplayName(), nil
}

// This job is used only for iDevices or for macos devices that don't use any
// setup experience items (software installs, script exec) - see
// appleMDMArgs.UseWorkerDeviceRelease. Otherwise releasing devices is now done
// via the orbit endpoint /setup_experience/status that is polled by a swift
// dialog UI window during the setup process, and automatically releases the
// device once all pending setup tasks are done.
func (a *AppleMDM) runPostDEPReleaseDevice(ctx context.Context, args appleMDMArgs) error {
	// Edge cases:
	//   - if the device goes offline for a long time, should we go ahead and
	//   release after a while?
	//   - if some commands/profiles failed (a final state), should we go ahead
	//   and release?
	//   - if the device keeps moving team, or profiles keep being added/removed
	//   from its team, it's possible that its profiles will never settle and
	//   always have pending statuses. Same as going offline, should we release
	//   after a while?
	//
	// We opted "yes" to all those, and we want to release after a few minutes,
	// not hours, so we'll allow only a couple retries.

	const (
		maxWaitTime         = 15 * time.Minute
		minAttempts         = 10
		maxAttempts         = 30
		nextAttemptMinDelay = 30 * time.Second
	)

	args.ReleaseDeviceAttempt++
	if args.ReleaseDeviceStartedAt == nil {
		now := time.Now().UTC()
		args.ReleaseDeviceStartedAt = &now
	}

	a.Log.DebugContext(ctx,
		fmt.Sprintf("awaiting commands %v and profiles to settle for host %s", args.EnrollmentCommands, args.HostUUID),
		"task", "runPostDEPReleaseDevice",
		"attempt", args.ReleaseDeviceAttempt,
		"started_at", args.ReleaseDeviceStartedAt.Format(time.RFC3339),
	)

	// if we've reached the minimum number of attempts and the maximum time to
	// wait, we release the device even if some commands or profiles are still
	// pending. We also release in case it reached the maximum number of
	// attempts, to prevent an issue with clock skew where the wait delay does
	// not appear to be reached.
	if (args.ReleaseDeviceAttempt >= minAttempts && time.Since(*args.ReleaseDeviceStartedAt) >= maxWaitTime) ||
		(args.ReleaseDeviceAttempt >= maxAttempts) {
		a.Log.InfoContext(ctx, "releasing device after too many attempts or too long wait", "host_uuid", args.HostUUID, "attempts", args.ReleaseDeviceAttempt)
		if err := a.Commander.DeviceConfigured(ctx, args.HostUUID, uuid.NewString()); err != nil {
			return ctxerr.Wrapf(ctx, err, "failed to enqueue DeviceConfigured command after %d attempts", args.ReleaseDeviceAttempt)
		}
		return nil
	}

	reenqueueTask := func() error {
		// re-enqueue the same job, but now
		// ReleaseDeviceAttempt/ReleaseDeviceStartedAt have been incremented/set,
		// and run it not before a delay so it doesn't run again until the next
		// worker cycle.
		_, err := QueueJobWithDelay(ctx, a.Datastore, appleMDMJobName, args, nextAttemptMinDelay)
		return err
	}

	// used to cross reference against the setup experience statuses below
	notNowCmdUUIDs := make(map[string]any)

	for _, cmdUUID := range args.EnrollmentCommands {
		if cmdUUID == "" {
			continue
		}

		res, err := a.Datastore.GetMDMAppleCommandResults(ctx, cmdUUID, args.HostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "failed to get MDM command results")
		}

		var completed bool
		for _, r := range res {
			if r.Status == fleet.MDMAppleStatusNotNow {
				notNowCmdUUIDs[cmdUUID] = ""
			}

			// succeeded or failed, it is done (final state). We also consider "NotNow"
			// as completed, as it means the device is not going to process that command
			// now, and we don't want to block the DEP device release because of that.
			if r.Status == fleet.MDMAppleStatusAcknowledged || r.Status == fleet.MDMAppleStatusError || r.Status == fleet.MDMAppleStatusNotNow || r.Status == fleet.MDMAppleStatusCommandFormatError {
				completed = true
				break
			}
		}

		if !completed {
			// DEP enrollment commands are not done being delivered to that device,
			// cannot release it now.
			if err := reenqueueTask(); err != nil {
				return fmt.Errorf("failed to re-enqueue task: %w", err)
			}
			return nil
		}
		a.Log.DebugContext(ctx,
			fmt.Sprintf("command %s has completed", cmdUUID),
			"task", "runPostDEPReleaseDevice",
		)
	}

	// all DEP-enrollment commands are done, check the host's profiles
	profs, err := a.Datastore.GetHostMDMAppleProfiles(ctx, args.HostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get host MDM profiles")
	}
	for _, prof := range profs {
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

		// if it has any pending profiles, then its profiles are not done being
		// delivered (installed or removed).
		if prof.Status == nil || *prof.Status == fleet.MDMDeliveryPending {
			if err := reenqueueTask(); err != nil {
				return fmt.Errorf("failed to re-enqueue task: %w", err)
			}
			return nil
		}
		a.Log.DebugContext(ctx,
			fmt.Sprintf("profile %s has been deployed", prof.Identifier),
			"task", "runPostDEPReleaseDevice",
		)
	}

	profilesMissingInstallation, err := a.Datastore.ListMDMAppleProfilesToInstall(ctx, args.HostUUID) // Get profiles that are missing to be installed on this host
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to list profiles missing installation")
	}
	profilesMissingInstallation = fleet.FilterOutUserScopedProfiles(profilesMissingInstallation)
	if !isMacOS(args.Platform) {
		profilesMissingInstallation = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(profilesMissingInstallation)
	}

	if len(profilesMissingInstallation) > 0 {
		a.Log.InfoContext(ctx, "re-enqueuing due to profiles missing installation", "host_uuid", args.HostUUID)
		// requeue the task if some profiles are still missing.
		if err := reenqueueTask(); err != nil {
			return ctxerr.Wrap(ctx, err, "failed to re-enqueue task")
		}
		return nil
	}

	if !isMacOS(args.Platform) {
		setupExperienceStatuses, err := a.Datastore.ListSetupExperienceResultsByHostUUID(ctx, args.HostUUID, ptr.ValOrZero(args.TeamID))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving setup experience status results for host pending DEP release")
		}
		for _, status := range setupExperienceStatuses {
			// skip items that had the command response of "NotNow" as those setup exp statuses will be pending/running
			// and we have decided to not block the device release for NotNow status so we dont want to reenqueue these.
			if status.NanoCommandUUID != nil {
				if _, ok := notNowCmdUUIDs[*status.NanoCommandUUID]; ok {
					continue
				}
			}

			if status.Status == fleet.SetupExperienceStatusPending || status.Status == fleet.SetupExperienceStatusRunning {
				a.Log.InfoContext(ctx, "re-enqueuing due to setup experience items still pending or running", "host_uuid", args.HostUUID, "status_id", status.ID)
				if err := reenqueueTask(); err != nil {
					return ctxerr.Wrap(ctx, err, "failed to re-enqueue task due to pending setup experience items")
				}
				return nil
			}
		}
	}

	// release the device
	a.Log.InfoContext(ctx, "releasing device, all DEP enrollment commands and profiles have completed", "host_uuid", args.HostUUID)
	if err := a.Commander.DeviceConfigured(ctx, args.HostUUID, uuid.NewString()); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to enqueue DeviceConfigured command")
	}
	return nil
}

func (a *AppleMDM) installFleetd(ctx context.Context, hostUUID string) (string, error) {
	manifestURL := fleetdbase.GetPKGManifestURL()
	cmdUUID := uuid.New().String()
	if err := a.Commander.InstallEnterpriseApplication(ctx, []string{hostUUID}, cmdUUID, manifestURL); err != nil {
		return "", err
	}
	a.Log.InfoContext(ctx, "sent command to install fleetd", "host_uuid", hostUUID)
	return cmdUUID, nil
}

func (a *AppleMDM) installSetupExperienceVPPAppsOnIosIpadOS(ctx context.Context, hostUUID string, teamID uint) ([]string, error) {
	statuses, err := a.Datastore.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving setup experience status results for next step")
	}

	var appsPending []*fleet.SetupExperienceStatusResult
	commandUUIDs := []string{}
	for _, status := range statuses {
		if err := status.IsValid(); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "invalid row")
		}

		switch {
		case status.VPPAppTeamID != nil:
			if status.Status == fleet.SetupExperienceStatusPending {
				appsPending = append(appsPending, status)
			}
		case status.SetupExperienceScriptID != nil, status.SoftwareInstallerID != nil:
			status.Status = fleet.SetupExperienceStatusFailure
			err = a.Datastore.UpdateSetupExperienceStatusResult(ctx, status)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "updating setup experience status result to failure")
			}
			// If we enqueued a non-VPP item for an iOS/iPadOS device, it likely a code bug
			a.Log.ErrorContext(ctx, "unexpected setup experience item for iOS/iPadOS device, only VPP apps are supported", "host_uuid", hostUUID, "status_id", status.ID)
		}
	}

	if len(appsPending) > 0 {
		// enqueue vpp apps
		// TODO Is there a better way to get a host by UUID? This is a somewhat "wide" search which feels unnecessary
		host, err := a.Datastore.HostByIdentifier(ctx, hostUUID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "retrieving host by UUID")
		}
		for _, app := range appsPending {
			vppAppID, err := app.VPPAppID()
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "constructing vpp app details for installation")
			}

			if app.SoftwareTitleID == nil {
				return nil, ctxerr.Errorf(ctx, "setup experience software title id missing from vpp app install request: %d", app.ID)
			}

			vppApp := &fleet.VPPApp{
				TitleID: *app.SoftwareTitleID,
				VPPAppTeam: fleet.VPPAppTeam{
					VPPAppID: *vppAppID,
				},
			}

			opts := fleet.HostSoftwareInstallOptions{
				SelfService:        false,
				ForSetupExperience: true,
			}

			cmdUUID, err := a.installSoftwareFromVPP(ctx, host, vppApp, true, opts)

			failedBeforeCommandSend := err != nil
			if err != nil {
				// if we get an error (e.g. no available licenses) while attempting to enqueue the
				// install, then we should immediately go to an error state so setup experience
				// isn't blocked.
				a.Log.ErrorContext(ctx, "got an error when attempting to enqueue VPP app install", "err", err, "adam_id", app.VPPAppAdamID)
				app.Status = fleet.SetupExperienceStatusFailure
				app.Error = ptr.String(err.Error())
			} else {
				app.NanoCommandUUID = &cmdUUID
				app.Status = fleet.SetupExperienceStatusRunning
				commandUUIDs = append(commandUUIDs, cmdUUID)
			}
			if err := a.Datastore.UpdateSetupExperienceStatusResult(ctx, app); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "updating setup experience with vpp install command uuid")
			}
			// Emit activity for the VPP app install failure, if one occurred
			if failedBeforeCommandSend && a.NewActivityFn != nil {
				failActivity := fleet.ActivityInstalledAppStoreApp{
					HostID:              host.ID,
					HostDisplayName:     host.DisplayName(),
					SoftwareTitle:       app.Name,
					AppStoreID:          ptr.ValOrZero(app.VPPAppAdamID),
					Status:              string(fleet.SoftwareInstallFailed),
					HostPlatform:        host.Platform,
					FromSetupExperience: true,
				}
				if actErr := a.NewActivityFn(ctx, nil, failActivity); actErr != nil {
					a.Log.WarnContext(ctx, "failed to create activity for VPP app install failure during setup experience", "err", actErr)
				}
			}
		}
	}

	return commandUUIDs, nil
}

func (a *AppleMDM) installSoftwareFromVPP(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, appleDevice bool, opts fleet.HostSoftwareInstallOptions) (string, error) {
	// Should not happen in the normal course of events but can happen in tests
	// and likely indicates things weren't initialized properly.
	if a.VPPInstaller == nil {
		return "", errors.New("VPP installer not configured")
	}
	token, err := a.VPPInstaller.GetVPPTokenIfCanInstallVPPApps(ctx, appleDevice, host)
	if err != nil {
		return "", err
	}

	return a.VPPInstaller.InstallVPPAppPostValidation(ctx, host, vppApp, token, opts)
}

func (a *AppleMDM) installBootstrapPackage(ctx context.Context, hostUUID string, teamID *uint) (string, error) {
	// GetMDMAppleBootstrapPackageMeta expects team id 0 for no team
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}
	meta, err := a.Datastore.GetMDMAppleBootstrapPackageMeta(ctx, tmID)
	if err != nil {
		var nfe fleet.NotFoundError
		if errors.As(err, &nfe) {
			a.Log.InfoContext(ctx, "unable to find a bootstrap package for DEP enrolled device, skipping installation", "host_uuid", hostUUID)
			return "", nil
		}

		return "", err
	}

	// Get CloudFront CDN signed URL if configured
	url := a.getSignedURL(ctx, meta)

	if url == "" {
		appCfg, err := a.Datastore.AppConfig(ctx)
		if err != nil {
			return "", err
		}

		url, err = meta.URL(appCfg.MDMUrl())
		if err != nil {
			return "", err
		}
	}

	manifest := appmanifest.NewFromSha(meta.Sha256, url)
	cmdUUID := uuid.New().String()
	err = a.Commander.InstallEnterpriseApplicationWithEmbeddedManifest(ctx, []string{hostUUID}, cmdUUID, manifest)
	if err != nil {
		return "", err
	}
	err = a.Datastore.RecordHostBootstrapPackage(ctx, cmdUUID, hostUUID)
	if err != nil {
		return "", err
	}
	a.Log.InfoContext(ctx, "sent command to install bootstrap package", "host_uuid", hostUUID)
	return cmdUUID, nil
}

func (a *AppleMDM) getSignedURL(ctx context.Context, meta *fleet.MDMAppleBootstrapPackage) string {
	var url string
	if a.BootstrapPackageStore != nil {
		pkgID := hex.EncodeToString(meta.Sha256)
		signedURL, err := a.BootstrapPackageStore.Sign(ctx, pkgID, fleet.BootstrapPackageSignedURLExpiry)
		switch {
		case errors.Is(err, fleet.ErrNotConfigured):
			// no CDN configured, fall back to the MDM URL
		case err != nil:
			// log the error but continue with the MDM URL
			a.Log.ErrorContext(ctx, "failed to sign bootstrap package URL", "err", err)
		default:
			exists, err := a.BootstrapPackageStore.Exists(ctx, pkgID)
			switch {
			case err != nil:
				// log the error but continue with the MDM URL
				a.Log.ErrorContext(ctx, "failed to check if bootstrap package exists", "err", err)
			case !exists:
				// log the error but continue with the MDM URL
				a.Log.ErrorContext(ctx, "bootstrap package does not exist in package store", "pkg_id", pkgID)
			default:
				url = signedURL
			}
		}
	}
	return url
}

// installProfilesForEnrollingHost installs all configuration profiles for the host immediately after enrollment
// to speed up the setup experience process. This runs before the reconciler cycle.
func (a *AppleMDM) installProfilesForEnrollingHost(ctx context.Context, hostUUID string) ([]string, error) {
	// Get all profiles that need to be installed for this host
	profilesToInstall, err := a.Datastore.ListMDMAppleProfilesToInstall(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing profiles to install for host")
	}

	profilesToInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(profilesToInstall)

	// Filter out user-scoped profiles as they require special handling
	profilesToInstall = fleet.FilterOutUserScopedProfiles(profilesToInstall)

	if len(profilesToInstall) == 0 {
		a.Log.InfoContext(ctx, "no profiles to install", "host_uuid", hostUUID)
		return nil, nil
	}

	a.Log.InfoContext(ctx, "installing profiles post-enrollment", "host_uuid", hostUUID, "profile_count", len(profilesToInstall))

	appConfig, err := a.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading app config")
	}

	hostProfiles := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(profilesToInstall))
	hostProfilesToInstallMap := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(profilesToInstall))
	installTargets := make(map[string]*fleet.CmdTarget, len(profilesToInstall))
	for _, profile := range profilesToInstall {
		target := &fleet.CmdTarget{
			CmdUUID:           uuid.NewString(),
			ProfileIdentifier: profile.ProfileIdentifier,
			ProfileName:       profile.ProfileName,
			EnrollmentIDs:     []string{hostUUID},
		}
		installTargets[profile.ProfileUUID] = target
		hostProfile := &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       profile.ProfileUUID,
			ProfileIdentifier: profile.ProfileIdentifier,
			ProfileName:       profile.ProfileName,
			HostUUID:          hostUUID,
			CommandUUID:       target.CmdUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            nil, // intentionally nil here, to avoid stuck pending, but we need to upsert before processing so inner code can match rows for failures
			Checksum:          profile.Checksum,
			SecretsUpdatedAt:  profile.SecretsUpdatedAt,
			Scope:             profile.Scope,
		}
		hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: hostUUID, ProfileUUID: profile.ProfileUUID}] = hostProfile
		hostProfiles = append(hostProfiles, hostProfile)
	}

	if err := a.Datastore.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk upsert host profiles before installation")
	}

	enqueueResult, err := apple_mdm.ProcessAndEnqueueProfiles(ctx, a.Datastore, a.Log, appConfig, a.Commander, installTargets, nil, hostProfilesToInstallMap, map[string]string{}, nil)
	if err != nil {
		return nil, err
	}

	// Build cmdUUID→profile index AFTER preprocessing has rewritten CommandUUIDs.
	profileByCmdUUID := make(map[string]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(hostProfilesToInstallMap))
	for _, hp := range hostProfilesToInstallMap {
		if hp.CommandUUID != "" {
			profileByCmdUUID[hp.CommandUUID] = hp
		}
	}

	// Log failures
	for cmdUUID, enqErr := range enqueueResult.FailedCmdUUIDs {
		if profile := profileByCmdUUID[cmdUUID]; profile != nil {
			a.Log.ErrorContext(ctx, "failed to install profile", "host_uuid", hostUUID, "profile_uuid", profile.ProfileUUID, "error", enqErr)
		}
	}

	// Collect successes for bulk upsert
	var cmdUUIDs []string
	var bulkPayloads []*fleet.MDMAppleBulkUpsertHostProfilePayload
	for _, cmdUUID := range enqueueResult.SucceededCmdUUIDs {
		if profile := profileByCmdUUID[cmdUUID]; profile != nil {
			profile.Status = &fleet.MDMDeliveryPending
			cmdUUIDs = append(cmdUUIDs, cmdUUID)
			bulkPayloads = append(bulkPayloads, profile)
		}
	}

	// Bulk update database to track all profile installations
	if len(bulkPayloads) > 0 {
		if err := a.Datastore.BulkUpsertMDMAppleHostProfiles(ctx, bulkPayloads); err != nil {
			a.Log.ErrorContext(ctx, "failed to bulk update profile statuses", "host_uuid", hostUUID, "error", err)
			// Continue even if database update fails - the commands were sent
		}
	}

	a.Log.InfoContext(ctx, "successfully queued profiles from apple mdm worker", "host_uuid", hostUUID, "profiles_sent", len(cmdUUIDs))

	// send a DeclarativeManagement command to start a sync, we don't block on DDM missing, and the declarations might not have been reconciled
	// We can come back to this if we want to include DDM declarations here in the future.
	declarativeManagementCmdUUID := uuid.NewString()
	if err := a.Commander.DeclarativeManagement(ctx, []string{hostUUID}, declarativeManagementCmdUUID); err != nil {
		a.Log.ErrorContext(ctx, "failed to send DeclarativeManagement command after installing profiles for enrolling host", "host_uuid", hostUUID, "error", err)
		// Make sure we return the profile commands even if DDM fails
		return cmdUUIDs, nil
	}
	cmdUUIDs = append(cmdUUIDs, declarativeManagementCmdUUID)

	return cmdUUIDs, nil
}

// QueueAppleMDMJob queues a apple_mdm job for one of the supported tasks, to
// be processed asynchronously via the worker.
func QueueAppleMDMJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	task AppleMDMTask,
	hostUUID string,
	platform string,
	teamID *uint,
	enrollReference string,
	useWorkerDeviceRelease bool,
	fromMDMMigration bool,
	enrollmentCommandUUIDs ...string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		appleMDMJobName, task,
		"host_uuid", hostUUID,
		"platform", platform,
		"with_enroll_reference", enrollReference != "",
		"from_mdm_migration", fromMDMMigration,
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	if len(enrollmentCommandUUIDs) > 0 {
		attrs = append(attrs, "enrollment_commands", fmt.Sprintf("%v", enrollmentCommandUUIDs))
	}
	logger.InfoContext(ctx, "queuing Apple MDM job", attrs...)

	args := &appleMDMArgs{
		Task:                   task,
		HostUUID:               hostUUID,
		TeamID:                 teamID,
		EnrollReference:        enrollReference,
		EnrollmentCommands:     enrollmentCommandUUIDs,
		Platform:               platform,
		UseWorkerDeviceRelease: useWorkerDeviceRelease,
		FromMDMMigration:       fromMDMMigration,
	}

	// the release device task is always added with a delay
	var delay time.Duration
	if task == AppleMDMPostDEPReleaseDeviceTask {
		delay = 30 * time.Second
	}
	job, err := QueueJobWithDelay(ctx, ds, appleMDMJobName, args, delay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	logger.DebugContext(ctx, "queued Apple MDM job", "job_id", job.ID)
	return nil
}

// sendManagedAccounts enqueues an AccountConfiguration command for an sso and/or
// a breakglass admin account.
func (a *AppleMDM) sendManagedAccounts(
	ctx context.Context,
	args *appleMDMArgs,
	ssoAccount *fleet.MDMIdPAccount,
	adminAccount *apple_mdm.AdminAccountConfig,
	lockPrimaryAccountInfo bool,
	cmdUUID string,
) error {
	var ssoConfig *apple_mdm.SSOAccountConfig
	if ssoAccount != nil {
		fullName, err := a.getIdPDisplayName(ctx, ssoAccount, *args)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting idp account display name")
		}
		a.Log.InfoContext(ctx, "setting username and fullname", "host_uuid", args.HostUUID)
		ssoConfig = &apple_mdm.SSOAccountConfig{
			FullName:               fullName,
			UserName:               ssoAccount.Username,
			LockPrimaryAccountInfo: lockPrimaryAccountInfo,
		}
	}

	if err := a.Commander.AccountConfiguration(ctx, []string{args.HostUUID}, cmdUUID, ssoConfig, adminAccount); err != nil {
		return ctxerr.Wrap(ctx, err, "sending AccountConfiguration command")
	}

	return nil
}
