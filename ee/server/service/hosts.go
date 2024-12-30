package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts fleet.ListOptions, includeCVSS bool) (*fleet.OSVersions, int, *fleet.PaginationMetadata, error) {
	// reuse OSVersions, but include premium options
	return svc.Service.OSVersions(ctx, teamID, platform, name, version, opts, true)
}

func (svc *Service) OSVersion(ctx context.Context, osID uint, teamID *uint, includeCVSS bool) (*fleet.OSVersion, *time.Time, error) {
	// reuse OSVersion, but include premium options
	return svc.Service.OSVersion(ctx, osID, teamID, true)
}

func (svc *Service) LockHost(ctx context.Context, hostID uint, viewPIN bool) (unlockPIN string, err error) {
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
	// requirement and is what happens for macOS platforms.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return "", err
	}

	// locking validations are based on the platform of the host
	switch host.FleetPlatform() {
	case "ios", "ipados":
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("host_id", "Can't lock iOS or iPadOS hosts. Use wipe instead."))
	case "darwin":
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
			if fleet.IsNotFound(err) {
				return "", ctxerr.Wrap(
					ctx, fleet.NewInvalidArgumentError("host_id", "Can't lock the host because it doesn't have MDM turned on."),
				)
			}
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
			ctx, fleet.NewInvalidArgumentError("host_id", "Host has pending lock request. The host will lock when it comes online."),
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

func (svc *Service) WipeHost(ctx context.Context, hostID uint) error {
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

	// wipe validations are based on the platform of the host, Windows and macOS
	// require MDM to be enabled and the host to be MDM-enrolled in Fleet. Linux
	// uses scripts, not MDM.
	var requireMDM bool
	switch host.FleetPlatform() {
	case "darwin", "ios", "ipados":
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
	return svc.enqueueWipeHostRequest(ctx, host, lockWipe)
}

func (svc *Service) enqueueLockHostRequest(ctx context.Context, host *fleet.Host, lockStatus *fleet.HostLockWipeStatus, viewPIN bool) (
	unlockPIN string, err error,
) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	if lockStatus.HostFleetPlatform == "darwin" {
		lockCommandUUID := uuid.NewString()
		if unlockPIN, err = svc.mdmAppleCommander.DeviceLock(ctx, host, lockCommandUUID); err != nil {
			return "", ctxerr.Wrap(ctx, err, "enqueuing lock request for darwin")
		}

		if err = svc.NewActivity(
			ctx,
			vc.User,
			fleet.ActivityTypeLockedHost{
				HostID:          host.ID,
				HostDisplayName: host.DisplayName(),
				ViewPIN:         viewPIN,
			},
		); err != nil {
			return "", ctxerr.Wrap(ctx, err, "create activity for darwin lock host request")
		}

		return unlockPIN, nil
	}

	script := windowsLockScript
	if lockStatus.HostFleetPlatform == "linux" {
		script = linuxLockScript
	}

	// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
	// part starting with the validation of the script (just in case), the checks
	// that we don't enqueue over the limit, etc. for any other important
	// validation we may add over there and that we bypass here by enqueueing the
	// script directly in the datastore layer.

	if err := svc.ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         host.ID,
		ScriptContents: string(script),
		UserID:         &vc.User.ID,
		SyncRequest:    false,
	}, host.FleetPlatform()); err != nil {
		return "", err
	}

	if err := svc.NewActivity(
		ctx,
		vc.User,
		fleet.ActivityTypeLockedHost{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
		},
	); err != nil {
		return "", ctxerr.Wrap(ctx, err, "create activity for lock host request")
	}

	return "", nil
}

func (svc *Service) enqueueUnlockHostRequest(ctx context.Context, host *fleet.Host, lockStatus *fleet.HostLockWipeStatus) (string, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	var unlockPIN string
	if lockStatus.HostFleetPlatform == "darwin" {
		// Record the unlock request time if it was not already recorded.
		// It should be always recorded, since the UnlockRequestedAt time is created after the lock command is acknowledged.
		// This code is left here to catch potential issues.
		if lockStatus.UnlockRequestedAt.IsZero() {
			if err := svc.ds.UnlockHostManually(ctx, host.ID, host.FleetPlatform(), time.Now().UTC()); err != nil {
				return "", err
			}
		}
		unlockPIN = lockStatus.UnlockPIN
	} else {
		script := windowsUnlockScript
		if lockStatus.HostFleetPlatform == "linux" {
			script = linuxUnlockScript
		}

		// TODO(mna): svc.RunHostScript should be refactored so that we can reuse the
		// part starting with the validation of the script (just in case), the checks
		// that we don't enqueue over the limit, etc. for any other important
		// validation we may add over there and that we bypass here by enqueueing the
		// script directly in the datastore layer.
		if err := svc.ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
			HostID:         host.ID,
			ScriptContents: string(script),
			UserID:         &vc.User.ID,
			SyncRequest:    false,
		}, host.FleetPlatform()); err != nil {
			return "", err
		}
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

func (svc *Service) enqueueWipeHostRequest(ctx context.Context, host *fleet.Host, wipeStatus *fleet.HostLockWipeStatus) error {
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
		wipeCmdUUID := uuid.NewString()
		wipeCmd := &fleet.MDMWindowsCommand{
			CommandUUID:  wipeCmdUUID,
			RawCommand:   []byte(fmt.Sprintf(windowsWipeCommand, wipeCmdUUID)),
			TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
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

// TODO(mna): ideally we'd embed the scripts from the scripts/mdm/windows/..
// and scripts/mdm/linux/.. directories where they currently exist, but this is
// not possible (not a Go package) and I don't know if those script locations
// are used elsewhere, so for now I just copied the contents under
// embedded_scripts directory. We'll have to make sure they are kept in sync,
// or better yet find a way to maintain a single copy.
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
					<LocURI>./Device/Vendor/MSFT/RemoteWipe/doWipeProtected</LocURI>
				</Target>
				<Meta>
					<Format xmlns="syncml:metinf">chr</Format>
					<Type>text/plain</Type>
				</Meta>
				<Data></Data>
			</Item>
		</Exec>`
)
