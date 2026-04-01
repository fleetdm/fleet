package recoverylock

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
)

// SendCommands is the cron job function that sends SetRecoveryLock MDM commands
// to hosts that need a recovery lock password.
//
// Note: SetRecoveryLock command results are handled in the MDM results handler
// (server/service/apple_mdm.go), which marks the password as verified upon acknowledgment.
func SendCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	return sendCommandsWithCommander(ctx, ds, commander, logger, newActivityFn)
}

func sendCommandsWithCommander(
	ctx context.Context,
	ds fleet.Datastore,
	commander Commander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	var result *multierror.Error

	// Restore hosts that were in "pending remove" state but feature was re-enabled.
	// This transitions them back to "verified install" to preserve the existing password.
	restored, err := ds.RestoreRecoveryLockForReenabledHosts(ctx)
	if err != nil {
		result = multierror.Append(result, ctxerr.Wrap(ctx, err, "restore recovery lock for re-enabled hosts"))
	} else if restored > 0 {
		logger.InfoContext(ctx, "restored recovery lock for re-enabled hosts", "count", restored)
	}

	// Handle SET password operations (hosts that need a recovery lock password)
	if err := sendSetCommands(ctx, ds, commander, logger); err != nil {
		result = multierror.Append(result, err)
	}

	// Handle CLEAR password operations (hosts that need their recovery lock cleared)
	if err := sendClearCommands(ctx, ds, commander, logger); err != nil {
		result = multierror.Append(result, err)
	}

	// Handle AUTO-ROTATION for viewed passwords (password viewed 1+ hour ago)
	if err := sendAutoRotationCommands(ctx, ds, commander, logger, newActivityFn); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func sendSetCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander Commander,
	logger *slog.Logger,
) error {
	hosts, err := ds.GetHostsForRecoveryLockAction(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need SetRecoveryLock")
		return nil
	}

	logger.InfoContext(ctx, "sending SetRecoveryLock commands", "count", len(hosts))

	// Generate passwords for all hosts upfront.
	// Passwords must be stored BEFORE enqueuing commands because they are injected
	// at delivery time by ExpandHostSecrets (which looks up by host UUID).
	passwords := make([]fleet.HostRecoveryLockPasswordPayload, 0, len(hosts))
	for _, hostUUID := range hosts {
		passwords = append(passwords, fleet.HostRecoveryLockPasswordPayload{
			HostUUID: hostUUID,
			Password: GeneratePassword(),
		})
	}

	// Store passwords with status='pending' atomically. This prevents the host from
	// being picked up again by the next cron run while we're enqueuing the command.
	// If enqueue fails, we reset the status to NULL so the host can be retried.
	if err := ds.SetHostsRecoveryLockPasswords(ctx, passwords); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set recovery lock passwords")
	}

	// Collect host UUIDs for enqueue.
	// The password is not in the command - a placeholder is used that will be
	// expanded at delivery time by ExpandHostSecrets.
	hostUUIDs := make([]string, 0, len(passwords))
	for _, p := range passwords {
		hostUUIDs = append(hostUUIDs, p.HostUUID)
	}

	// Enqueue a single command for all hosts. Each host gets their own queue entry
	// pointing to the same command, and ExpandHostSecrets injects the per-host
	// password at delivery time.
	cmdUUID := uuid.NewString()
	if err := commander.SetRecoveryLock(ctx, hostUUIDs, cmdUUID); err != nil {
		// Check if this is an APNs delivery error (command was persisted but push failed).
		// In this case, the command is already queued and will be delivered when the device
		// checks in, so we should NOT clear the pending status (which would cause duplicates).
		var apnsErr *apple_mdm.APNSDeliveryError
		if errors.As(err, &apnsErr) {
			// Command was persisted but push notification failed - log warning but don't fail.
			// The command will be delivered when the device next checks in.
			logger.WarnContext(ctx, "SetRecoveryLock commands enqueued but APNs push failed",
				"host_count", len(hostUUIDs),
				"command_uuid", cmdUUID,
				"error", err,
			)
			// Don't clear pending status - command is queued and will be processed
			return nil
		}

		// Persistence failed - reset status to NULL so hosts will be picked up again on next cron run.
		// The password is already stored, but a new one will be generated on retry (overwrites old).
		logger.ErrorContext(ctx, "failed to enqueue SetRecoveryLock commands",
			"host_count", len(hostUUIDs),
			"error", err,
		)
		if clearErr := ds.ClearRecoveryLockPendingStatus(ctx, hostUUIDs); clearErr != nil {
			logger.ErrorContext(ctx, "failed to clear recovery lock pending status after enqueue failure",
				"host_count", len(hostUUIDs),
				"error", clearErr,
			)
			err = multierror.Append(err, clearErr)
		}
		return ctxerr.Wrap(ctx, err, "enqueue SetRecoveryLock commands")
	}

	logger.InfoContext(ctx, "sent SetRecoveryLock commands",
		"host_count", len(hostUUIDs),
		"command_uuid", cmdUUID,
	)

	return nil
}

func sendClearCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander Commander,
	logger *slog.Logger,
) error {
	hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock clear action")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need ClearRecoveryLock")
		return nil
	}

	logger.InfoContext(ctx, "sending ClearRecoveryLock commands", "count", len(hosts))

	// Enqueue clear command. The CurrentPassword placeholder will be expanded at
	// delivery time by ExpandHostSecrets (which looks up by host UUID).
	cmdUUID := uuid.NewString()
	if err := commander.ClearRecoveryLock(ctx, hosts, cmdUUID); err != nil {
		var apnsErr *apple_mdm.APNSDeliveryError
		if errors.As(err, &apnsErr) {
			// Command was persisted but push notification failed - log warning but don't fail.
			logger.WarnContext(ctx, "ClearRecoveryLock commands enqueued but APNs push failed",
				"host_count", len(hosts),
				"command_uuid", cmdUUID,
				"error", err,
			)
			return nil
		}

		// Persistence failed - reset status to NULL so hosts will be picked up again.
		logger.ErrorContext(ctx, "failed to enqueue ClearRecoveryLock commands",
			"host_count", len(hosts),
			"error", err,
		)
		if clearErr := ds.ClearRecoveryLockPendingStatus(ctx, hosts); clearErr != nil {
			logger.ErrorContext(ctx, "failed to clear recovery lock pending status after enqueue failure",
				"host_count", len(hosts),
				"error", clearErr,
			)
			err = multierror.Append(err, clearErr)
		}
		return ctxerr.Wrap(ctx, err, "enqueue ClearRecoveryLock commands")
	}

	logger.InfoContext(ctx, "sent ClearRecoveryLock commands",
		"host_count", len(hosts),
		"command_uuid", cmdUUID,
	)

	return nil
}

func sendAutoRotationCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander Commander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	hosts, err := ds.GetHostsForAutoRotation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for auto rotation")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need auto-rotation")
		return nil
	}

	logger.InfoContext(ctx, "performing auto-rotation for viewed passwords", "count", len(hosts))

	var result *multierror.Error
	for _, host := range hosts {
		newPassword := GeneratePassword()

		// Initiate rotation - stores pending password and validates eligibility
		if err := ds.InitiateRecoveryLockRotation(ctx, host.HostUUID, newPassword); err != nil {
			// Check for benign race conditions where host state changed between
			// GetHostsForAutoRotation and now (e.g., manual rotation started,
			// password removed, host deleted, etc.)
			if fleet.IsNotFound(err) ||
				errors.Is(err, fleet.ErrRecoveryLockRotationPending) ||
				errors.Is(err, fleet.ErrRecoveryLockNotEligible) {
				logger.DebugContext(ctx, "host lost eligibility for auto-rotation",
					"host_uuid", host.HostUUID,
					"error", err,
				)
				continue
			}

			logger.ErrorContext(ctx, "failed to initiate auto-rotation",
				"host_uuid", host.HostUUID,
				"error", err,
			)
			result = multierror.Append(result, err)
			continue
		}

		// Enqueue RotateRecoveryLock command
		cmdUUID := uuid.NewString()
		if err := commander.RotateRecoveryLock(ctx, host.HostUUID, cmdUUID); err != nil {
			var apnsErr *apple_mdm.APNSDeliveryError
			if errors.As(err, &apnsErr) {
				// Command was persisted but push notification failed - log activity and continue.
				// The command will be retried when the device checks in.
				logAutoRotationActivity(ctx, logger, newActivityFn, host)
				logger.WarnContext(ctx, "auto-rotation command enqueued but APNs push failed",
					"host_uuid", host.HostUUID,
					"command_uuid", cmdUUID,
					"error", err,
				)
				continue
			}

			// Persistence failed - clear pending rotation so host can be retried
			logger.ErrorContext(ctx, "failed to enqueue auto-rotation command",
				"host_uuid", host.HostUUID,
				"error", err,
			)
			if clearErr := ds.ClearRecoveryLockRotation(ctx, host.HostUUID); clearErr != nil {
				logger.ErrorContext(ctx, "failed to clear pending rotation after enqueue failure",
					"host_uuid", host.HostUUID,
					"error", clearErr,
				)
				result = multierror.Append(result, clearErr)
			}
			result = multierror.Append(result, err)
			continue
		}

		// Log activity for auto-rotation (Fleet-initiated)
		logAutoRotationActivity(ctx, logger, newActivityFn, host)

		logger.DebugContext(ctx, "sent auto-rotation command",
			"host_uuid", host.HostUUID,
			"command_uuid", cmdUUID,
		)
	}

	return result.ErrorOrNil()
}

// logAutoRotationActivity logs the rotation activity for auto-rotations.
// It uses the same activity type as manual rotations but marks it as Fleet-initiated.
func logAutoRotationActivity(
	ctx context.Context,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
	host fleet.HostAutoRotationInfo,
) {
	if newActivityFn == nil {
		return
	}

	if err := newActivityFn(ctx, nil, fleet.ActivityTypeRotatedHostRecoveryLockPassword{
		HostID:          host.HostID,
		HostDisplayName: host.DisplayName,
		FleetInitiated:  true,
	}); err != nil {
		logger.WarnContext(ctx, "auto-rotation: failed to create activity",
			"host_uuid", host.HostUUID,
			"err", err,
		)
	}
}
