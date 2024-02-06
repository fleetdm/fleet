package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts fleet.ListOptions, includeCVSS bool) (*fleet.OSVersions, int, *fleet.PaginationMetadata, error) {
	// reuse OSVersions, but include premium options
	return svc.Service.OSVersions(ctx, teamID, platform, name, version, opts, true)
}

func (svc *Service) OSVersion(ctx context.Context, osID uint, teamID *uint, includeCVSS bool) (*fleet.OSVersion, *time.Time, error) {
	// reuse OSVersion, but include premium options
	return svc.Service.OSVersion(ctx, osID, teamID, true)
}

func (svc *Service) LockHost(ctx context.Context, hostID uint) error {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lite")
	}

	// Authorize again with team loaded now that we have the host's team_id.
	// Authorize as "execute mdm_command", which is the correct access
	// requirement and is what happens for macOS platforms.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// locking validations are based on the platform of the host
	switch host.FleetPlatform() {
	case "darwin":
		// on macOS, the lock command requires the host to be MDM-enrolled in Fleet
		hostMDM, err := svc.ds.GetHostMDM(ctx, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host MDM information")
		}
		if !hostMDM.IsFleetEnrolled() {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Can't lock the host because it doesn't have MDM turned on."))
		}

	case "windows", "linux":
		// on windows and linux, a script is used to lock the host so scripts must
		// be enabled
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get app config")
		}
		if appCfg.ServerSettings.ScriptsDisabled {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Can't lock host because running scripts is disabled in organization settings."))
		}

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	// if there's a lock, unlock or wipe action pending, do not accept the lock
	// request. A quick initial check is to look if ActionsSuspended is true,
	// meaning that a lock, unlock or wipe is pending, otherwise no such action
	// is pending.
	lockWipe, err := svc.ds.GetHostLockWipeStatus(ctx, host.ID, host.FleetPlatform())
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}
	// lockWipe contains the lock status (i.e. a reference to the lock MDM
	// command or script which may be pending, succeeded or failed, or nil if not
	// locked nor pending), the wipe status (same, but for wipe), and the unlock
	// status (nil if not pending, otherwise a reference to the unlock script or
	// PIN number for macOS).
	switch {
	case lockWipe.IsPendingLock():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending lock request. The host will lock when it comes online."))
	case lockWipe.IsPendingUnlock():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending unlock request. Host cannot be locked again until unlock is complete."))
	case lockWipe.IsPendingWipe():
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending wipe request. Cannot process lock requests once host is wiped."))
	case lockWipe.IsLocked():
		// TODO(mna): succeed quietly, returning the current unlock pin for macos?
	}

	// all good, go ahead with queuing the lock request.
	panic("unimplemented")
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
	case "darwin":
		// all good, no need to check if MDM enrolled, will validate later that it
		// is currently locked.

	case "windows", "linux":
		// on windows and linux, a script is used to lock the host so scripts must
		// be enabled
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "get app config")
		}
		if appCfg.ServerSettings.ScriptsDisabled {
			return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Can't unlock host because running scripts is disabled in organization settings."))
		}

	default:
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", fmt.Sprintf("Unsupported host platform: %s", host.Platform)))
	}

	lockWipe, err := svc.ds.GetHostLockWipeStatus(ctx, host.ID, host.FleetPlatform())
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}
	switch {
	case lockWipe.IsPendingLock():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending lock request. Host cannot be unlocked until lock is complete."))
	case lockWipe.IsPendingUnlock():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending unlock request. The host will unlock when it comes online."))
	case lockWipe.IsPendingWipe():
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending wipe request. Cannot process unlock requests once host is wiped."))
	case lockWipe.IsUnlocked():
		// TODO(mna): error, already unlocked, or succeed quietly? If so, what to return for macOS?
	}
	panic("unimplemented")
}
