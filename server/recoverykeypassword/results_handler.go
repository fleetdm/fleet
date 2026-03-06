package recoverykeypassword

import (
	"context"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/groob/plist"
)

// recoveryLockResult wraps mdm.CommandResults to implement fleet.MDMCommandResults
type recoveryLockResult struct {
	cmdResult *mdm.CommandResults
}

func (r *recoveryLockResult) Raw() []byte      { return r.cmdResult.Raw }
func (r *recoveryLockResult) UUID() string     { return r.cmdResult.CommandUUID }
func (r *recoveryLockResult) HostUUID() string { return r.cmdResult.Identifier() }

// NewRecoveryLockResult wraps an mdm.CommandResults to implement fleet.MDMCommandResults
func NewRecoveryLockResult(cmdResult *mdm.CommandResults) fleet.MDMCommandResults {
	return &recoveryLockResult{cmdResult: cmdResult}
}

// verifyRecoveryLockResponse represents the MDM response for VerifyRecoveryLock command.
type verifyRecoveryLockResponse struct {
	Status           string `plist:"Status"`
	PasswordVerified bool   `plist:"PasswordVerified"`
}

// NewVerifyRecoveryLockResultsHandler processes VerifyRecoveryLock command results.
// When a VerifyRecoveryLock command is acknowledged with PasswordVerified=true,
// it marks the recovery lock as verified. If PasswordVerified=false, it marks as failed.
// NotNow status is ignored (device will retry).
func NewVerifyRecoveryLockResultsHandler(
	rkpDS Datastore,
	logger *slog.Logger,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, results fleet.MDMCommandResults) error {
		// Check if this is a verification command from our cron job (by prefix)
		if !strings.HasPrefix(results.UUID(), VerifyRecoveryLockCommandPrefix) {
			// Not a VerifyRecoveryLock command from our cron job, skip
			return nil
		}

		// Get host ID from command UUID
		hostID, err := rkpDS.GetHostIDByVerifyCommandUUID(ctx, results.UUID())
		if err != nil {
			if fleet.IsNotFound(err) {
				// Not a VerifyRecoveryLock command from our cron job, skip
				return nil
			}
			return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: get host id by verify command uuid")
		}

		// Get the underlying result to access status and error chain
		rlResult, ok := results.(*recoveryLockResult)
		if !ok {
			return ctxerr.New(ctx, "VerifyRecoveryLock handler: unexpected results type")
		}

		status := rlResult.cmdResult.Status
		logger.DebugContext(ctx, "VerifyRecoveryLock command result received",
			"host_id", hostID,
			"host_uuid", results.HostUUID(),
			"command_uuid", results.UUID(),
			"status", status,
		)

		switch status {
		case fleet.MDMAppleStatusAcknowledged:
			// Parse the response to check PasswordVerified field
			var response verifyRecoveryLockResponse
			if err := plist.Unmarshal(rlResult.cmdResult.Raw, &response); err != nil {
				return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: unmarshal response")
			}

			if response.PasswordVerified {
				// Password verified successfully
				if err := rkpDS.SetRecoveryLockVerified(ctx, hostID); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock verified")
				}

				logger.InfoContext(ctx, "VerifyRecoveryLock acknowledged, password verified",
					"host_id", hostID,
					"host_uuid", results.HostUUID(),
					"command_uuid", results.UUID(),
				)
			} else {
				// Password verification failed - password doesn't match
				if err := rkpDS.SetRecoveryLockFailed(ctx, hostID, "password verification failed: password does not match"); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock failed")
				}

				logger.WarnContext(ctx, "VerifyRecoveryLock acknowledged but password verification failed",
					"host_id", hostID,
					"host_uuid", results.HostUUID(),
					"command_uuid", results.UUID(),
				)
			}

		case fleet.MDMAppleStatusNotNow:
			// Device is busy, will retry later. Leave status as verifying.
			logger.DebugContext(ctx, "VerifyRecoveryLock returned NotNow, device will retry",
				"host_id", hostID,
				"host_uuid", results.HostUUID(),
				"command_uuid", results.UUID(),
			)

		case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
			// Command error
			errorMsg := apple_mdm.FmtErrorChain(rlResult.cmdResult.ErrorChain)
			if err := rkpDS.SetRecoveryLockFailed(ctx, hostID, errorMsg); err != nil {
				return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock failed")
			}

			logger.WarnContext(ctx, "VerifyRecoveryLock command failed",
				"host_id", hostID,
				"host_uuid", results.HostUUID(),
				"command_uuid", results.UUID(),
				"error", errorMsg,
			)
		}

		return nil
	}
}
