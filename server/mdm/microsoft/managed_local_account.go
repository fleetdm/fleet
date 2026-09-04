package microsoft_mdm

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/hashicorp/go-multierror"
)

// SendManagedLocalAccountRotationRequests rotates the Windows managed local admin (`_fleetadmin`) password for hosts
// whose auto_rotate_at has elapsed. It is the Windows counterpart to apple_mdm.SendManagedLocalAccountRotationCommands
// and runs on the same schedule.
//
// There is no command to enqueue: recording the request is the whole of the work, and the host's next orbit config
// check-in asks it to re-provision the account, which resets the password and escrows the new one. A benign race (the
// row's eligibility flipped since the SELECT, the host unenrolled) is debug-logged and skipped rather than failing the
// cron iteration.
//
// Activity logging mirrors the macOS path:
//   - rows with initiated_by_fleet=1 (view-driven) log with FleetInitiated=true
//   - rows with initiated_by_fleet=0 (a manual request awaiting its turn) skip logging, because the manual path
//     already logged the activity with the requesting user as actor
func SendManagedLocalAccountRotationRequests(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	hosts, err := ds.GetWindowsManagedLocalAccountsForAutoRotation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get windows managed local accounts for auto rotation")
	}
	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no windows managed local accounts due for rotation")
		return nil
	}
	logger.InfoContext(ctx, "requesting windows managed local account password rotations", "count", len(hosts))

	var result *multierror.Error
	for _, host := range hosts {
		if err := ds.InitiateWindowsManagedLocalAccountRotation(ctx, host.HostUUID); err != nil {
			if fleet.IsNotFound(err) ||
				errors.Is(err, fleet.ErrManagedLocalAccountRotationPending) ||
				errors.Is(err, fleet.ErrManagedLocalAccountNotEligible) {
				logger.DebugContext(ctx, "windows managed local account no longer eligible for rotation",
					"host_uuid", host.HostUUID, "err", err)
				continue
			}
			logger.ErrorContext(ctx, "windows managed local account rotation",
				"host_uuid", host.HostUUID, "err", err)
			result = multierror.Append(result, err)
			continue
		}

		logManagedLocalAccountRotationActivity(ctx, logger, newActivityFn, host)
		logger.DebugContext(ctx, "requested windows managed local account rotation", "host_uuid", host.HostUUID)
	}

	return result.ErrorOrNil()
}

// logManagedLocalAccountRotationActivity logs the rotation activity ONLY for view-driven rows
// (initiated_by_fleet=1). A row with initiated_by_fleet=0 belongs to a manual request whose activity was already
// recorded with the requesting user as actor; re-logging here would double-count it.
func logManagedLocalAccountRotationActivity(
	ctx context.Context,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
	host fleet.HostManagedLocalAccountWindowsRotationInfo,
) {
	if !host.InitiatedByFleet || newActivityFn == nil {
		return
	}
	if err := newActivityFn(ctx, nil, fleet.ActivityTypeRotatedManagedLocalAccountPassword{
		HostID:          host.HostID,
		HostDisplayName: host.DisplayName,
		FleetInitiated:  true,
	}); err != nil {
		logger.WarnContext(ctx, "windows managed local account rotation: failed to create activity",
			"host_uuid", host.HostUUID, "err", err)
	}
}
