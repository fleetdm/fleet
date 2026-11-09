package apple_mdm

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

// ReleaseDevice sends DeviceConfigured to a host held in Setup Assistant by the setup experience,
// then clears its awaiting-configuration flag, which stops fleetd from polling the status endpoint.
// Polls after the flag is cleared send nothing. The flag is cleared after the send, not before, so a
// failure in between leaves it set for a later poll to retry rather than leaving the host unreleased.
func ReleaseDevice(ctx context.Context, ds fleet.Datastore, commander fleet.MDMAppleCommandIssuer, logger *slog.Logger, hostUUID string) error {
	// Primary: a lagging replica would still show a flag that the previous poll just cleared.
	awaiting, err := ds.GetHostAwaitingConfiguration(ctxdb.RequirePrimary(ctx, true), hostUUID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get host awaiting configuration")
	}
	if !awaiting {
		return nil
	}

	if err := SendDeviceConfigured(ctx, commander, logger, hostUUID); err != nil {
		return err
	}
	if err := ds.SetHostAwaitingConfiguration(ctx, hostUUID, false); err != nil {
		return ctxerr.Wrap(ctx, err, "unset host awaiting configuration")
	}
	return nil
}

// SendDeviceConfigured enqueues DeviceConfigured for the host. A failed push after the command is
// queued is logged rather than returned: the device still gets the command at its next check-in, and
// a caller retrying on the error would queue another one.
func SendDeviceConfigured(ctx context.Context, commander fleet.MDMAppleCommandIssuer, logger *slog.Logger, hostUUID string) error {
	if err := commander.DeviceConfigured(ctx, hostUUID, uuid.NewString()); err != nil {
		if _, isNotifErr := errors.AsType[*NotificationFailedError](err); !isNotifErr {
			return ctxerr.Wrap(ctx, err, "enqueue DeviceConfigured command")
		}
		logger.ErrorContext(ctx, "sending push notification for DeviceConfigured", "host_uuid", hostUUID, "err", err)
	}
	return nil
}
