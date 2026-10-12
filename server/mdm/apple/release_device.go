package apple_mdm

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

// ReleaseDevice sends DeviceConfigured to a host held in Setup Assistant by the setup experience. It
// clears the host's awaiting-configuration flag first, which stops fleetd from polling the status
// endpoint, and only the call that clears it sends the command, so repeated or concurrent polls send
// it once. Clearing before enqueueing means a failed write sends nothing.
func ReleaseDevice(ctx context.Context, ds fleet.Datastore, commander fleet.MDMAppleCommandIssuer, logger *slog.Logger, hostUUID string) error {
	cleared, err := ds.UnsetHostAwaitingConfiguration(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unset host awaiting configuration")
	}
	if !cleared {
		return nil
	}

	if err := SendDeviceConfigured(ctx, commander, logger, hostUUID); err != nil {
		// Nothing was queued, so put the flag back for the next poll to retry.
		if restoreErr := ds.SetHostAwaitingConfiguration(ctx, hostUUID, true); restoreErr != nil {
			logger.ErrorContext(ctx, "restoring host awaiting configuration after DeviceConfigured enqueue failed", "host_uuid", hostUUID, "err", restoreErr)
		}
		return err
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
