package recoverykeypassword

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

// MDMCommander defines the MDM operations needed by the recovery lock reconciler.
type MDMCommander interface {
	// EnqueueCommand enqueues a raw MDM command for the given host UUIDs.
	EnqueueCommand(ctx context.Context, hostUUIDs []string, rawCommand string) error
	// SendNotifications sends APNs push notifications to wake up devices.
	SendNotifications(ctx context.Context, hostUUIDs []string) error
}

// ReconcileRecoveryLockPasswords is the main cron job function that manages recovery lock passwords.
// It performs three tasks:
// 1. Checks for acknowledged SetRecoveryLock commands and sends VerifyRecoveryLock commands
// 2. Re-pushes verifying hosts (where VerifyRecoveryLock hasn't been acknowledged)
// 3. Sends SetRecoveryLock commands to hosts that need a recovery lock password
func ReconcileRecoveryLockPasswords(
	ctx context.Context,
	rkpDS Datastore,
	commander MDMCommander,
	logger *slog.Logger,
) error {
	// Step 1: Process pending SetRecoveryLock commands (check for acknowledged/failed)
	if err := processPendingSetCommands(ctx, rkpDS, commander, logger); err != nil {
		// Log error but continue with other steps
		logger.ErrorContext(ctx, "error processing pending SetRecoveryLock commands", "error", err)
	}

	// Step 2: Send SetRecoveryLock to hosts that need it
	if err := sendSetRecoveryLockCommands(ctx, rkpDS, commander, logger); err != nil {
		return ctxerr.Wrap(ctx, err, "send SetRecoveryLock commands")
	}

	return nil
}

// processPendingSetCommands checks for hosts with pending SetRecoveryLock commands
// and either sends VerifyRecoveryLock (if acknowledged) or marks as failed (if errored).
func processPendingSetCommands(
	ctx context.Context,
	rkpDS Datastore,
	commander MDMCommander,
	logger *slog.Logger,
) error {
	pendingHosts, err := rkpDS.GetPendingRecoveryLockHosts(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get pending recovery lock hosts")
	}

	if len(pendingHosts) == 0 {
		return nil
	}

	logger.DebugContext(ctx, "processing pending SetRecoveryLock commands", "count", len(pendingHosts))

	for _, host := range pendingHosts {
		switch host.SetCommandStatus {
		case fleet.MDMAppleStatusAcknowledged:
			// SetRecoveryLock was acknowledged, send VerifyRecoveryLock
			if err := sendVerifyRecoveryLock(ctx, rkpDS, commander, logger, host); err != nil {
				logger.ErrorContext(ctx, "failed to send VerifyRecoveryLock",
					"host_id", host.HostID,
					"host_uuid", host.HostUUID,
					"error", err,
				)
				continue
			}

		case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
			// SetRecoveryLock failed, mark as failed so it will be retried
			errorMsg := host.SetCommandErrorInfo
			if errorMsg == "" {
				errorMsg = "SetRecoveryLock command failed"
			}
			if err := rkpDS.SetRecoveryLockFailed(ctx, host.HostID, errorMsg); err != nil {
				logger.ErrorContext(ctx, "failed to mark recovery lock as failed",
					"host_id", host.HostID,
					"host_uuid", host.HostUUID,
					"error", err,
				)
				continue
			}
			logger.WarnContext(ctx, "SetRecoveryLock command failed",
				"host_id", host.HostID,
				"host_uuid", host.HostUUID,
				"command_uuid", host.SetCommandUUID,
				"error", errorMsg,
			)

		default:
			// No result yet (empty string) or other status, skip for now
			continue
		}
	}

	return nil
}

// sendVerifyRecoveryLock sends a VerifyRecoveryLock command for a host
// after the SetRecoveryLock command has been acknowledged.
func sendVerifyRecoveryLock(
	ctx context.Context,
	rkpDS Datastore,
	commander MDMCommander,
	logger *slog.Logger,
	host HostPendingRecoveryLock,
) error {
	// Get the stored password
	rkp, err := rkpDS.GetHostRecoveryKeyPassword(ctx, host.HostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host recovery key password")
	}

	// Generate verification command UUID with prefix
	cmdUUID := VerifyRecoveryLockCommandPrefix + uuid.NewString()

	// Send VerifyRecoveryLock command
	rawCmd := VerifyRecoveryLockCommand(cmdUUID, rkp.Password)
	if err := commander.EnqueueCommand(ctx, []string{host.HostUUID}, string(rawCmd)); err != nil {
		return ctxerr.Wrap(ctx, err, "send VerifyRecoveryLock command")
	}

	// Update status to verifying with the verify command UUID
	if err := rkpDS.SetRecoveryLockVerifying(ctx, host.HostID, cmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock verifying")
	}

	logger.DebugContext(ctx, "sent VerifyRecoveryLock command",
		"host_id", host.HostID,
		"host_uuid", host.HostUUID,
		"command_uuid", cmdUUID,
	)

	return nil
}

// sendSetRecoveryLockCommands sends SetRecoveryLock commands to hosts that need them.
func sendSetRecoveryLockCommands(
	ctx context.Context,
	rkpDS Datastore,
	commander MDMCommander,
	logger *slog.Logger,
) error {
	hosts, err := rkpDS.GetHostsForRecoveryLockAction(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need SetRecoveryLock")
		return nil
	}

	logger.InfoContext(ctx, "sending SetRecoveryLock commands", "count", len(hosts))

	for _, host := range hosts {
		if err := processHostForSet(ctx, rkpDS, commander, logger, host); err != nil {
			// Log error but continue with other hosts
			logger.ErrorContext(ctx, "failed to process host for SetRecoveryLock",
				"host_id", host.HostID,
				"host_uuid", host.HostUUID,
				"error", err,
			)
			continue
		}
	}

	return nil
}

func processHostForSet(
	ctx context.Context,
	rkpDS Datastore,
	commander MDMCommander,
	logger *slog.Logger,
	host HostRecoveryLockAction,
) error {
	// Generate and store password (this creates/updates the record)
	password, err := rkpDS.SetHostRecoveryKeyPassword(ctx, host.HostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set host recovery key password")
	}

	// Generate command UUID
	cmdUUID := uuid.NewString()

	// Send SetRecoveryLock command
	rawCmd := SetRecoveryLockCommand(cmdUUID, password)
	if err := commander.EnqueueCommand(ctx, []string{host.HostUUID}, string(rawCmd)); err != nil {
		return ctxerr.Wrap(ctx, err, "send SetRecoveryLock command")
	}

	// Update status to pending with the command UUID
	if err := rkpDS.SetRecoveryLockPending(ctx, host.HostID, cmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock pending")
	}

	logger.DebugContext(ctx, "sent SetRecoveryLock command",
		"host_id", host.HostID,
		"host_uuid", host.HostUUID,
		"command_uuid", cmdUUID,
	)

	return nil
}
