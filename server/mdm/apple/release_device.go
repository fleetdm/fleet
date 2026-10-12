package apple_mdm

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

// ReleaseDevice sends DeviceConfigured to a host held in Setup Assistant. It is safe to call
// repeatedly (the setup experience status endpoint is polled every few seconds): it sends
// nothing while a previous DeviceConfigured is still waiting for a result.
func ReleaseDevice(ctx context.Context, ds fleet.Datastore, commander fleet.MDMAppleCommandIssuer, logger *slog.Logger, hostUUID string) error {
	pending, acknowledged, err := ds.GetHostDeviceConfiguredStatus(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get DeviceConfigured status")
	}

	switch {
	case acknowledged:
		// The device is already released. The DeviceConfigured result handler clears this flag,
		// but if it failed the host would keep polling and asking for another release.
		if err := ds.SetHostAwaitingConfiguration(ctx, hostUUID, false); err != nil {
			return ctxerr.Wrap(ctx, err, "unset host awaiting configuration")
		}
		return nil
	case pending:
		return nil
	}

	if err := commander.DeviceConfigured(ctx, hostUUID, uuid.NewString()); err != nil {
		// The command is queued, so the device still gets it at its next check-in.
		if _, isNotifErr := errors.AsType[*NotificationFailedError](err); !isNotifErr {
			return ctxerr.Wrap(ctx, err, "enqueue DeviceConfigured command")
		}
		logger.ErrorContext(ctx, "sending push notification for DeviceConfigured", "host_uuid", hostUUID, "err", err)
	}
	return nil
}
