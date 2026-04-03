// Package recoverylock provides recovery lock password management for macOS devices.
//
// This package consolidates all recovery lock password functionality including:
// - Password generation (GeneratePassword)
// - Cron job processing for SET/CLEAR/ROTATE operations (SendCommands)
// - Result handling from MDM command acknowledgments (NewResultsHandler)
//
// Note: This is a domain module within the MDM structure, not a fully isolated
// bounded context. It has dependencies on MDM infrastructure components.
//
// # Types
//
// The following types from the fleet package are part of this module's API:
//
//   - fleet.HostRecoveryLockPassword - password record for a host
//   - fleet.HostRecoveryLockPasswordPayload - payload for storing passwords
//   - fleet.HostRecoveryLockRotationStatus - rotation state for a host
//   - fleet.HostAutoRotationInfo - minimal host data for auto-rotation logging
//   - fleet.HostMDMRecoveryLockPassword - status representation for API responses
//   - fleet.RecoveryLockStatus - enum for status values
//   - fleet.ErrRecoveryLockRotationPending - rotation already in progress
//   - fleet.ErrRecoveryLockNotEligible - host not eligible for rotation
//
// # Datastore Methods
//
// The following fleet.Datastore methods are part of this module's contract:
//
//   - SetHostsRecoveryLockPasswords
//   - GetHostRecoveryLockPassword
//   - GetHostRecoveryLockPasswordStatus
//   - GetHostsForRecoveryLockAction
//   - RestoreRecoveryLockForReenabledHosts
//   - SetRecoveryLockVerified
//   - SetRecoveryLockFailed
//   - ClearRecoveryLockPendingStatus
//   - ClaimHostsForRecoveryLockClear
//   - DeleteHostRecoveryLockPassword
//   - GetRecoveryLockOperationType
//   - InitiateRecoveryLockRotation
//   - CompleteRecoveryLockRotation
//   - FailRecoveryLockRotation
//   - ClearRecoveryLockRotation
//   - GetRecoveryLockRotationStatus
//   - HasPendingRecoveryLockRotation
//   - ResetRecoveryLockForRetry
//   - MarkRecoveryLockPasswordViewed
//   - GetHostsForAutoRotation
package recoverylock

import "context"

// Commander defines the interface for sending recovery lock commands.
// This interface is implemented by MDMAppleCommander and allows for testing.
type Commander interface {
	SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	RotateRecoveryLock(ctx context.Context, hostUUID string, cmdUUID string) error
}
