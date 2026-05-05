package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	opts.IncludeCriticalVulnerabilitiesCount = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts fleet.ListOptions, _ bool,
	maxVulnerabilities *int,
) (*fleet.OSVersions, int, *fleet.PaginationMetadata, error) {
	// reuse OSVersions, but include premium options
	return svc.Service.OSVersions(ctx, teamID, platform, name, version, opts, true, maxVulnerabilities)
}

func (svc *Service) OSVersion(ctx context.Context, osID uint, teamID *uint, _ bool, maxVulnerabilities *int) (*fleet.OSVersion, *time.Time, error) {
	// reuse OSVersion, but include premium options
	return svc.Service.OSVersion(ctx, osID, teamID, true, maxVulnerabilities)
}

func (svc *Service) LockHost(ctx context.Context, hostID uint, viewPIN bool) (unlockPIN string, err error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return "", err
	}
	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host lite")
	}

	// Authorize again with team loaded now that we have the host's team_id.
	// Authorize as "execute mdm_command", which is the correct access
	// requirement and is what happens for macOS platforms.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return "", err
	}

	// locking validations are based on the platform of the host
	switch host.FleetPlatform() {
	case "darwin", "ios", "ipados":
		if host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "On (personal)" {
			return "", &fleet.BadRequestError{
				Message: fleet.CantLockPersonalHostsMessage,
			}
		}
		if host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "On (manual)" &&
			(host.FleetPlatform() == "ios" || host.FleetPlatform() == "ipados") {
			return "", &fleet.BadRequestError{
				Message: fleet.CantLockManualIOSIpadOSHostsMessage,
			}
		}
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			if errors.Is(err, fleet.ErrMDMNotConfigured) {
				err = fleet.NewInvalidArgumentError("host_id", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			}
			return "", ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
		}

		// on macOS, the lock command requires the host to be MDM-enrolled in Fleet
		connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
		}
		if !connected {
			return "", ctxerr.Wrap(
				ctx, fleet.NewInvalidArgumentError("host_id", "Can't lock the host because it doesn't have MDM turned on."),
			)
		}

	case "windows", "linux":
		if host.FleetPlatform() == "windows" {
			if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
				if errors.Is(err, fleet.ErrMDMNotConfigured) {
					err = fleet.NewInvalidArgumentError("host_id", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
				}
				return "", ctxerr.Wrap(ctx, err, "check windows MDM enabled")
			}
		}
		hostOrbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
		switch {
		case err != nil:
			// If not found, then do nothing. We do not know if this host has scripts enabled or not
			if !fleet.IsNotFound(err) {
				return "", ctxerr.Wrap(ctx, err, "get host orbit info")
			}
		case hostOrbitInfo.ScriptsEnabled != nil && !*hostOrbitInfo.ScriptsEnabled:
			return "", ctxerr.Wrap(
				ctx, fleet.NewInvalidArgumentError(
					"host_id", "Couldn't lock host. To lock, deploy the fleetd agent with --enable-scripts and refetch host vitals.",
				),
			)
		}

	default:
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	// if there's a lock, unlock or wipe action pending, do not accept the lock
	// request.
	lockWipe, err := svc.ds.GetHostLockWipeStatus(ctx, host)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}

	switch {
	case lockWipe.IsPendingLock():
		return "", ctxerr.Wrap(
			ctx, fleet.NewInvalidArgumentError(
				"host_id", "Host has pending lock request. Host cannot be locked again until lock is complete.",
			),
		)
	case lockWipe.IsPendingUnlock():
		return "", ctxerr.Wrap(
			ctx, fleet.NewInvalidArgumentError(
				"host_id", "Host has pending unlock request. Host cannot be locked again until unlock is complete.",
			),
		)
	case lockWipe.IsPendingWipe():
		return "", ctxerr.Wrap(
			ctx,
			fleet.NewInvalidArgumentError("host_id", "Host has pending wipe request. Cannot process lock requests once host is wiped."),
		)
	case lockWipe.IsWiped():
		return "", ctxerr.Wrap(
			ctx, fleet.NewInvalidArgumentError("host_id", "Host is wiped. Cannot process lock requests once host is wiped."),
		)
	case lockWipe.IsLocked():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host is already locked.").WithStatus(http.StatusConflict))
	}

	// all good, go ahead with queuing the lock request.
	return svc.enqueueLockHostRequest(ctx, host, lockWipe, viewPIN)
}

func (svc *Service) UnlockHost(ctx context.Context, hostID uint) (string, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return "", err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host lite")
	}

	// Authorize again with team loaded now that we have the host's team_id.
	// Authorize as "execute mdm_command", which is the correct access
	// requirement.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return "", err
	}

	// locking validations are based on the platform of the host
	switch host.FleetPlatform() {
	case "darwin", "ios", "ipados":
		// all good, no need to check if MDM enrolled, will validate later that it
		// is currently locked.

	case "windows", "linux":
		// on Windows and Linux, a script is used to unlock the host so scripts must
		// be enabled on the host
		if host.FleetPlatform() == "windows" {
			if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
				if errors.Is(err, fleet.ErrMDMNotConfigured) {
					err = fleet.NewInvalidArgumentError("host_id", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
				}
				return "", ctxerr.Wrap(ctx, err, "check windows MDM enabled")
			}
		}
		hostOrbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
		switch {
		case err != nil:
			// If not found, then do nothing. We do not know if this host has scripts enabled or not
			if !fleet.IsNotFound(err) {
				return "", ctxerr.Wrap(ctx, err, "get host orbit info")
			}
		case hostOrbitInfo.ScriptsEnabled != nil && !*hostOrbitInfo.ScriptsEnabled:
			return "", ctxerr.Wrap(
				ctx, fleet.NewInvalidArgumentError(
					"host_id", "Couldn't unlock host. To unlock, deploy the fleetd agent with --enable-scripts and refetch host vitals.",
				),
			)
		}

	default:
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	lockWipe, err := svc.ds.GetHostLockWipeStatus(ctx, host)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}
	switch {
	case lockWipe.IsPendingLock():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending lock request. Host cannot be unlocked until lock is complete."))
	case lockWipe.IsPendingUnlock():
		// MacOS machines are unlocked by typing the PIN into the machine. "Unlock" in this case
		// should just return the PIN as many times as needed.
		// Breaking here will fall through to call enqueueUnLockHostRequest which will return the PIN.
		if host.FleetPlatform() == "darwin" {
			break
		}
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending unlock request. The host will unlock when it comes online."))
	case lockWipe.IsPendingWipe():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending wipe request. Cannot process unlock requests once host is wiped."))
	case lockWipe.IsWiped():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host is wiped. Cannot process unlock requests once host is wiped."))
	case lockWipe.IsUnlocked():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host is already unlocked.").WithStatus(http.StatusConflict))
	}

	// all good, go ahead with queuing the unlock request.
	return svc.enqueueUnlockHostRequest(ctx, host, lockWipe)
}

func (svc *Service) WipeHost(ctx context.Context, hostID uint, metadata *fleet.MDMWipeMetadata) error {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lite")
	}

	// Authorize again with team loaded now that we have the host's team_id.
	// Authorize as "execute mdm_command", which is the correct access
	// requirement and is what happens for macOS platforms.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// wipe validations are based on the platform of the host, Windows and macOS
	// require MDM to be enabled and the host to be MDM-enrolled in Fleet. Linux
	// uses scripts, not MDM.
	var requireMDM bool
	switch host.FleetPlatform() {
	case "darwin", "ios", "ipados":
		if host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "On (personal)" {
			return &fleet.BadRequestError{
				Message: fleet.CantWipePersonalHostsMessage,
			}
		}
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			if errors.Is(err, fleet.ErrMDMNotConfigured) {
				err = fleet.NewInvalidArgumentError("host_id", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			}
			return ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
		}
		requireMDM = true

	case "windows":
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			if errors.Is(err, fleet.ErrMDMNotConfigured) {
				err = fleet.NewInvalidArgumentError("host_id", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			}
			return ctxerr.Wrap(ctx, err, "check windows MDM enabled")
		}
		requireMDM = true

	case "linux":
		// on linux, a script is used to wipe the host so scripts must be enabled on the host
		hostOrbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
		switch {
		case err != nil:
			// If not found, then do nothing. We do not know if this host has scripts enabled or not
			if !fleet.IsNotFound(err) {
				return ctxerr.Wrap(ctx, err, "get host orbit info")
			}
		case hostOrbitInfo.ScriptsEnabled != nil && !*hostOrbitInfo.ScriptsEnabled:
			return ctxerr.Wrap(
				ctx, fleet.NewInvalidArgumentError(
					"host_id", "Couldn't wipe host. To wipe, deploy the fleetd agent with --enable-scripts and refetch host vitals.",
				),
			)
		}

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	if requireMDM {
		// the wipe command requires the host to be MDM-enrolled in Fleet
		connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
		}
		if !connected {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Can't wipe the host because it doesn't have MDM turned on."))
		}
	}

	// validations based on host's actions status (pending lock, unlock, wipe)
	lockWipe, err := svc.ds.GetHostLockWipeStatus(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}
	switch {
	case lockWipe.IsPendingLock():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending lock request. Host cannot be wiped until lock is complete."))
	case lockWipe.IsPendingUnlock():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending unlock request. Host cannot be wiped until unlock is complete."))
	case lockWipe.IsPendingWipe():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending wipe request. The host will be wiped when it comes online."))
	case lockWipe.IsLocked():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host is locked. Host cannot be wiped until it is unlocked."))
	case lockWipe.IsWiped():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host is already wiped.").WithStatus(http.StatusConflict))
	}

	// all good, go ahead with queuing the wipe request.
	return svc.enqueueWipeHostRequest(ctx, host, lockWipe, metadata)
}

func (svc *Service) enqueueLockHostRequest(ctx context.Context, host *fleet.Host, lockStatus *fleet.HostLockWipeStatus, viewPIN bool) (
	unlockPIN string, err error,
) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	activity := fleet.ActivityTypeLockedHost{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
	}

	switch lockStatus.HostFleetPlatform {
	case "darwin":
		lockCommandUUID := uuid.NewString()
		if unlockPIN, err = svc.mdmAppleCommander.DeviceLock(ctx, host, lockCommandUUID); err != nil {
			return "", ctxerr.Wrap(ctx, err, "enqueuing lock request for macOS")
		}
		activity.ViewPIN = viewPIN
	case "ios", "ipados":
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "get app config")
		}
		lockCommandUUID := uuid.NewString()
		if err := svc.mdmAppleCommander.EnableLostMode(ctx, host, lockCommandUUID, appCfg.OrgInfo.OrgName); err != nil {
			return "", ctxerr.Wrap(ctx, err, "enabling lost mode for iOS/iPadOS")
		}
	case "windows":
		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(windowsLockScript),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return "", err
		}
	case "linux":
		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(linuxLockScript),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return "", err
		}
	}

	if err := svc.NewActivity(
		ctx,
		vc.User,
		activity,
	); err != nil {
		return "", ctxerr.Wrap(ctx, err, "create activity for lock host request")
	}

	return unlockPIN, nil
}

func (svc *Service) enqueueUnlockHostRequest(ctx context.Context, host *fleet.Host, lockStatus *fleet.HostLockWipeStatus) (unlockPIN string, err error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	switch lockStatus.HostFleetPlatform {
	case "darwin":
		// Record the unlock request time if it was not already recorded.
		// It should be always recorded, since the UnlockRequestedAt time is created after the lock command is acknowledged.
		// This code is left here to catch potential issues.
		if lockStatus.UnlockRequestedAt.IsZero() {
			if err := svc.ds.UnlockHostManually(ctx, host.ID, host.FleetPlatform(), time.Now().UTC()); err != nil {
				return "", err
			}
		}
		unlockPIN = lockStatus.UnlockPIN
	case "ios", "ipados":
		err := svc.mdmAppleCommander.DisableLostMode(ctx, host, uuid.NewString())
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "disabling lost mode for iOS/iPadOS")
		}
	case "windows":
		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(windowsUnlockScript),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return "", err
		}
	case "linux":
		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(linuxUnlockScript),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return "", err
		}
	default:
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	if err := svc.NewActivity(
		ctx,
		vc.User,
		fleet.ActivityTypeUnlockedHost{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
			HostPlatform:    host.Platform,
		},
	); err != nil {
		return "", ctxerr.Wrap(ctx, err, "create activity for unlock host request")
	}

	return unlockPIN, nil
}

func (svc *Service) enqueueWipeHostRequest(
	ctx context.Context,
	host *fleet.Host,
	wipeStatus *fleet.HostLockWipeStatus,
	metadata *fleet.MDMWipeMetadata,
) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	switch wipeStatus.HostFleetPlatform {
	case "darwin", "ios", "ipados":
		wipeCommandUUID := uuid.NewString()
		if err := svc.mdmAppleCommander.EraseDevice(ctx, host, wipeCommandUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "enqueuing wipe request for darwin")
		}

	case "windows":
		// default wipe type
		wipeType := fleet.MDMWindowsWipeTypeDoWipeProtected
		if metadata != nil && metadata.Windows != nil {
			wipeType = metadata.Windows.WipeType
			svc.logger.DebugContext(ctx, "Windows host wipe request", "wipe_type", wipeType.String())
		}
		wipeCmdUUID := uuid.NewString()
		wipeCmd := &fleet.MDMWindowsCommand{
			CommandUUID:  wipeCmdUUID,
			RawCommand:   []byte(fmt.Sprintf(windowsWipeCommand, wipeCmdUUID, wipeType.String())),
			TargetLocURI: fmt.Sprintf("./Device/Vendor/MSFT/RemoteWipe/%s", wipeType.String()),
		}
		if err := svc.ds.WipeHostViaWindowsMDM(ctx, host, wipeCmd); err != nil {
			return ctxerr.Wrap(ctx, err, "enqueuing wipe request for windows")
		}

	case "linux":
		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.WipeHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(linuxWipeScript),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return err
		}
	}

	if err := svc.NewActivity(
		ctx,
		vc.User,
		fleet.ActivityTypeWipedHost{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for wipe host request")
	}
	return nil
}

func (svc *Service) RotateRecoveryLockPassword(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host")
	}

	// Authorize again with team loaded now that we have the host's team_id.
	// Authorize as "execute mdm_command", which is the correct access requirement.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// Validate: must be Apple Silicon Mac (macOS with ARM CPU)
	if !host.IsAppleSilicon() {
		return &fleet.BadRequestError{
			Message: "Recovery lock password rotation is only supported on Apple Silicon Macs.",
		}
	}

	// Validate: must be MDM enrolled in Fleet
	connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet MDM")
	}
	if !connected {
		return &fleet.BadRequestError{
			Message: "Host must be enrolled in Fleet MDM to rotate the recovery lock password.",
		}
	}

	// Check if recovery lock password is enabled for this team/no-team
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}

	recoveryLockEnabled := false
	if host.TeamID != nil {
		team, err := svc.ds.TeamLite(ctx, *host.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team")
		}
		recoveryLockEnabled = team.Config.MDM.EnableRecoveryLockPassword
	} else {
		recoveryLockEnabled = appCfg.MDM.EnableRecoveryLockPassword.Value
	}

	if !recoveryLockEnabled {
		return &fleet.BadRequestError{
			Message: "Recovery lock password is not enabled for this host's team.",
		}
	}

	// Get the current rotation status
	rotationStatus, err := svc.ds.GetRecoveryLockRotationStatus(ctx, host.UUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Host does not have a recovery lock password to rotate.",
			}
		}
		return ctxerr.Wrap(ctx, err, "get recovery lock rotation status")
	}

	// Validate: must have an existing password
	if !rotationStatus.HasPassword {
		return &fleet.BadRequestError{
			Message: "Host does not have a recovery lock password to rotate.",
		}
	}

	// Validate: not already rotating
	if rotationStatus.HasPendingRotation {
		return &fleet.ConflictError{
			Message: "Recovery lock password rotation is already in progress for this host.",
		}
	}

	// Validate: must be in install operation (not remove)
	if rotationStatus.OperationType == string(fleet.MDMOperationTypeRemove) {
		return &fleet.BadRequestError{
			Message: "Cannot rotate recovery lock password while a clear operation is in progress.",
		}
	}

	// Validate: must have status verified or failed (not pending or NULL)
	status := ""
	if rotationStatus.Status != nil {
		status = *rotationStatus.Status
	}
	if status != string(fleet.MDMDeliveryVerified) && status != string(fleet.MDMDeliveryFailed) {
		return &fleet.BadRequestError{
			Message: "Cannot rotate recovery lock password while an operation is pending.",
		}
	}

	// Generate new password
	newPassword := apple_mdm.GenerateRecoveryLockPassword()

	// Store pending rotation
	if err := svc.ds.InitiateRecoveryLockRotation(ctx, host.UUID, newPassword); err != nil {
		return ctxerr.Wrap(ctx, err, "initiate recovery lock rotation")
	}

	// Enqueue MDM command
	cmdUUID := uuid.NewString()
	if err := svc.mdmAppleCommander.RotateRecoveryLock(ctx, host.UUID, cmdUUID); err != nil {
		// Only clear the pending rotation if the enqueue itself failed.
		// If it's an APNS delivery error, the command was successfully enqueued
		// and will be delivered when the device checks in.
		var apnsErr *apple_mdm.APNSDeliveryError
		if !errors.As(err, &apnsErr) {
			_ = svc.ds.ClearRecoveryLockRotation(ctx, host.UUID)
		}
		return ctxerr.Wrap(ctx, err, "enqueue recovery lock rotation command")
	}

	// Log activity
	vc, ok := viewer.FromContext(ctx)
	if ok {
		if err := svc.NewActivity(
			ctx,
			vc.User,
			fleet.ActivityTypeRotatedHostRecoveryLockPassword{
				HostID:          host.ID,
				HostDisplayName: host.DisplayName(),
			},
		); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for rotate recovery lock password")
		}
	}

	return nil
}

var (
	//go:embed embedded_scripts/windows_lock.ps1
	windowsLockScript []byte
	//go:embed embedded_scripts/windows_unlock.ps1
	windowsUnlockScript []byte
	//go:embed embedded_scripts/linux_lock.sh
	linuxLockScript []byte
	//go:embed embedded_scripts/linux_unlock.sh
	linuxUnlockScript []byte
	//go:embed embedded_scripts/linux_wipe.sh
	linuxWipeScript []byte

	windowsWipeCommand = `
		<Exec>
			<CmdID>%s</CmdID>
			<Item>
				<Target>
					<LocURI>./Device/Vendor/MSFT/RemoteWipe/%s</LocURI>
				</Target>
				<Meta>
					<Format xmlns="syncml:metinf">chr</Format>
					<Type>text/plain</Type>
				</Meta>
				<Data></Data>
			</Item>
		</Exec>`
)

func (svc *Service) GetHostManagedAccountPassword(ctx context.Context, hostID uint) (*fleet.HostManagedLocalAccountPassword, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host lite")
	}
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}
	if !fleet.IsMacOSPlatform(host.Platform) {
		return nil, &fleet.BadRequestError{
			Message: "Host is not a macOS device.",
		}
	}

	acct, err := svc.ds.GetHostManagedLocalAccountStatus(ctx, host.UUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, &fleet.BadRequestError{
				Message: "Host does not have a managed account.",
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "get host managed account status")
	}
	// Behavior change vs #43381: gate on password_available rather than
	// status == 'verified'. A row whose status is 'pending' due to a recent view
	// (or a deferred rotation waiting on UUID capture) still has a valid password
	// and should be viewable. Only 'failed' or a missing password block access.
	if !acct.PasswordAvailable {
		return nil, &fleet.BadRequestError{
			Message: "Host's managed account password is not available.",
		}
	}

	pwd, err := svc.ds.GetHostManagedLocalAccountPassword(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host managed account password")
	}

	// Surface the rotation lifecycle alongside the password so the modal can
	// render the auto-rotate / pending-rotation banner on first open without a
	// separate host-details refetch round-trip.
	pwd.PendingRotation = acct.PendingRotation

	// Start the auto-rotation timer (no-op for views inside the existing window)
	// and capture the resulting deadline. notFound here means a rotation is
	// currently in flight (pending_encrypted_password IS NOT NULL) — the
	// password is still readable and the in-flight rotation will refresh it, so
	// we leave AutoRotateAt nil and rely on PendingRotation for the UI signal.
	rotateAt, err := svc.ds.MarkManagedLocalAccountPasswordViewed(ctx, host.UUID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "mark managed local account password viewed")
	}
	if err == nil {
		pwd.AutoRotateAt = &rotateAt
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeViewedManagedLocalAccount{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create viewed managed local account activity")
	}

	return pwd, nil
}

// RotateManagedLocalAccountPassword rotates the macOS managed local admin
// (`_fleetadmin`) password. When account_uuid is captured we generate a new
// password, stage it as a pending rotation, and enqueue SetAutoAdminPassword.
// When account_uuid is missing we record a deferred rotation that the cron
// will fulfill once the UUID arrives via osquery — the user-actor activity
// is still logged immediately (the cron must NOT re-log it for these rows).
//
// Idempotency: if a rotation is already in flight (pending_encrypted_password
// IS NOT NULL) we return success without enqueueing again. The pending rotation
// will land via the device ack and the next user-initiated rotate (if needed)
// can proceed afterwards.
func (svc *Service) RotateManagedLocalAccountPassword(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lite")
	}
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}
	if !fleet.IsMacOSPlatform(host.Platform) {
		return &fleet.BadRequestError{Message: "Host is not a macOS device."}
	}

	acct, err := svc.ds.GetHostManagedLocalAccountStatus(ctx, host.UUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{Message: "Host does not have a managed account."}
		}
		return ctxerr.Wrap(ctx, err, "get host managed account status")
	}
	if !acct.PasswordAvailable {
		return &fleet.BadRequestError{Message: "Couldn’t rotate managed local account password. Please try again."}
	}
	if acct.PendingRotation {
		return &fleet.BadRequestError{Message: "Managed local account password rotation is already in progress for this host."}
	}

	accountUUID, err := svc.ds.GetManagedLocalAccountUUID(ctx, host.UUID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get managed local account uuid")
	}

	// Defer if UUID isn't yet captured. The activity is logged with the calling
	// user as actor at click time, and the cron will execute the rotation later
	// without re-logging because initiated_by_fleet=0 on this row.
	if accountUUID == nil || *accountUUID == "" {
		if err := svc.logRotateManagedLocalAccountActivity(ctx, host, false); err != nil {
			return err
		}
		if err := svc.ds.MarkManagedLocalAccountRotationDeferred(ctx, host.UUID); err != nil {
			return ctxerr.Wrap(ctx, err, "mark managed local account rotation deferred")
		}
		return nil
	}

	newPassword := apple_mdm.GenerateManagedAccountPassword()
	hashPlist, err := apple_mdm.GenerateSaltedSHA512PBKDF2Hash(newPassword)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generate managed local account password hash")
	}

	cmdUUID := uuid.NewString()
	if err := svc.ds.InitiateManagedLocalAccountRotation(ctx, host.UUID, newPassword, cmdUUID); err != nil {
		// Race against the cron: a rotation snuck in between our
		// PendingRotation check and now. Treat the same as the idempotent
		// short-circuit above.
		if errors.Is(err, fleet.ErrManagedLocalAccountRotationPending) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "initiate managed local account rotation")
	}

	if err := svc.mdmAppleCommander.SetAutoAdminPassword(ctx, host.UUID, *accountUUID, hashPlist, cmdUUID); err != nil {
		// APNs delivery error: the command is persisted in nano and will be
		// delivered on the next checkin. Keep pending state and log the
		// activity (consistent with the manual recovery-lock path).
		var apnsErr *apple_mdm.APNSDeliveryError
		if errors.As(err, &apnsErr) {
			if logErr := svc.logRotateManagedLocalAccountActivity(ctx, host, false); logErr != nil {
				return logErr
			}
			return ctxerr.Wrap(ctx, err, "enqueue managed local account rotation command")
		}
		// Persistence failure: the command never landed. Roll back pending
		// state and DON'T log the activity so the user can retry cleanly.
		if clearErr := svc.ds.ClearManagedLocalAccountRotation(ctx, host.UUID); clearErr != nil {
			svc.logger.ErrorContext(ctx, "failed to clear managed local account pending rotation after enqueue error",
				"host_uuid", host.UUID, "err", clearErr)
		}
		return ctxerr.Wrap(ctx, err, "enqueue managed local account rotation command")
	}

	return svc.logRotateManagedLocalAccountActivity(ctx, host, false)
}

func (svc *Service) logRotateManagedLocalAccountActivity(ctx context.Context, host *fleet.Host, fleetInitiated bool) error {
	vc, ok := viewer.FromContext(ctx)
	var actor *fleet.User
	if ok {
		actor = vc.User
	}
	if err := svc.NewActivity(ctx, actor, fleet.ActivityTypeRotatedManagedLocalAccountPassword{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		FleetInitiated:  fleetInitiated,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create rotated managed local account activity")
	}
	return nil
}
