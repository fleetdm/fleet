package recoverylock

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// Result wraps mdm.CommandResults to implement fleet.MDMCommandResults
type Result struct {
	CmdResult *mdm.CommandResults
}

func (r *Result) Raw() []byte      { return r.CmdResult.Raw }
func (r *Result) UUID() string     { return r.CmdResult.CommandUUID }
func (r *Result) HostUUID() string { return r.CmdResult.UDID } // SetRecoveryLock is device-only, UDID is always present

// NewResult wraps an mdm.CommandResults to implement fleet.MDMCommandResults
func NewResult(cmdResult *mdm.CommandResults) fleet.MDMCommandResults {
	return &Result{CmdResult: cmdResult}
}

// NewResultsHandler processes SetRecoveryLock command results.
// It handles SET (install), CLEAR (remove), and ROTATE operations:
// - SET: When acknowledged, marks the recovery lock as verified. On error, marks as failed.
// - CLEAR: When acknowledged, deletes the recovery lock password record. On error, marks as failed.
// - ROTATE: When acknowledged, moves pending password to active. On error, marks rotation as failed.
func NewResultsHandler(
	ds fleet.Datastore,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, results fleet.MDMCommandResults) error {
		// Get the underlying result to access status and error chain
		rlResult, ok := results.(*Result)
		if !ok {
			return ctxerr.New(ctx, "SetRecoveryLock handler: unexpected results type")
		}

		hostUUID := results.HostUUID()
		status := rlResult.CmdResult.Status

		// Check if this is a rotation (has pending password)
		hasPendingRotation, err := ds.HasPendingRecoveryLockRotation(ctx, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: check pending rotation")
		}

		if hasPendingRotation {
			// This is a rotation result
			logger.DebugContext(ctx, "SetRecoveryLock rotation result received",
				"host_uuid", hostUUID,
				"command_uuid", results.UUID(),
				"status", status,
			)

			switch status {
			case fleet.MDMAppleStatusAcknowledged:
				// Rotation succeeded - move pending password to active
				if err := ds.CompleteRecoveryLockRotation(ctx, hostUUID); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: complete rotation")
				}

				logger.InfoContext(ctx, "RotateRecoveryLock acknowledged, password rotated",
					"host_uuid", hostUUID,
				)

			case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
				errorMsg := apple_mdm.FmtErrorChain(rlResult.CmdResult.ErrorChain)
				if errorMsg == "" {
					errorMsg = "RotateRecoveryLock command failed"
				}
				if err := ds.FailRecoveryLockRotation(ctx, hostUUID, errorMsg); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: fail rotation")
				}
				logger.WarnContext(ctx, "RotateRecoveryLock command failed",
					"host_uuid", hostUUID,
					"error", errorMsg,
				)
			}

			return nil
		}

		// Get the operation type to determine if this was a SET or CLEAR operation
		opType, err := ds.GetRecoveryLockOperationType(ctx, hostUUID)
		if err != nil {
			// If the record doesn't exist, it may have been deleted already - nothing to do
			if fleet.IsNotFound(err) {
				logger.DebugContext(ctx, "SetRecoveryLock result received but no password record exists",
					"host_uuid", hostUUID,
					"status", status,
				)
				return nil
			}
			return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: get operation type")
		}

		logger.DebugContext(ctx, "SetRecoveryLock command result received",
			"host_uuid", hostUUID,
			"command_uuid", results.UUID(),
			"status", status,
			"operation_type", opType,
		)

		switch status {
		case fleet.MDMAppleStatusAcknowledged:
			if opType == fleet.MDMOperationTypeRemove {
				// CLEAR succeeded - delete the password record
				if err := ds.DeleteHostRecoveryLockPassword(ctx, hostUUID); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: delete recovery lock password")
				}
				logger.InfoContext(ctx, "ClearRecoveryLock acknowledged, password record deleted",
					"host_uuid", hostUUID,
				)
			} else {
				// SET succeeded - mark as verified
				if err := ds.SetRecoveryLockVerified(ctx, hostUUID); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock verified")
				}

				// Get host info for activity logging - don't fail the operation if this fails
				var hostID uint
				var displayName string
				host, err := ds.HostLiteByIdentifier(ctx, hostUUID)
				if err != nil {
					logger.WarnContext(ctx, "SetRecoveryLock handler: failed to get host for activity logging",
						"host_uuid", hostUUID,
						"err", err,
					)
				} else {
					hostID = host.ID
					displayName = host.Hostname

					// Log the activity only if we could identify the host (fleet-initiated via WasFromAutomation)
					if err := newActivityFn(ctx, nil, fleet.ActivityTypeSetHostRecoveryLockPassword{
						HostID:          hostID,
						HostDisplayName: displayName,
					}); err != nil {
						logger.WarnContext(ctx, "SetRecoveryLock handler: failed to create activity",
							"host_uuid", hostUUID,
							"err", err,
						)
					}
				}

				logger.InfoContext(ctx, "SetRecoveryLock acknowledged, marked verified",
					"host_uuid", hostUUID,
					"host_id", hostID,
				)
			}

		case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
			errorMsg := apple_mdm.FmtErrorChain(rlResult.CmdResult.ErrorChain)
			if errorMsg == "" {
				if opType == fleet.MDMOperationTypeRemove {
					errorMsg = "ClearRecoveryLock command failed"
				} else {
					errorMsg = "SetRecoveryLock command failed"
				}
			}

			if opType == fleet.MDMOperationTypeRemove {
				// CLEAR operation failed
				// Command format errors are terminal - command is malformed and won't succeed on retry.
				// Password mismatch errors are also terminal - requires admin intervention.
				if rlResult.CmdResult.Status == fleet.MDMAppleStatusCommandFormatError ||
					apple_mdm.IsRecoveryLockPasswordMismatchError(rlResult.CmdResult.ErrorChain) {
					if err := ds.SetRecoveryLockFailed(ctx, hostUUID, errorMsg); err != nil {
						return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock failed")
					}
					logger.WarnContext(ctx, "ClearRecoveryLock failed with terminal error",
						"host_uuid", hostUUID,
						"error", errorMsg,
					)
				} else {
					// Transient error - reset to install/verified for retry on next cron cycle
					if err := ds.ResetRecoveryLockForRetry(ctx, hostUUID); err != nil {
						return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: reset recovery lock for retry")
					}
					logger.InfoContext(ctx, "ClearRecoveryLock failed with transient error, will retry",
						"host_uuid", hostUUID,
						"error", errorMsg,
					)
				}
			} else {
				// SET operation failed - mark as failed
				if err := ds.SetRecoveryLockFailed(ctx, hostUUID, errorMsg); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock failed")
				}
				logger.WarnContext(ctx, "SetRecoveryLock command failed",
					"host_uuid", hostUUID,
					"error", errorMsg,
				)
			}
		}

		return nil
	}
}
