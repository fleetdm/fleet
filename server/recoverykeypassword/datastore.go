package recoverykeypassword

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// VerifyRecoveryLockCommandPrefix is the prefix used for VerifyRecoveryLock MDM command UUIDs.
const VerifyRecoveryLockCommandPrefix = "VERIFY-RECOVERY-LOCK-"

// HostRecoveryKeyPassword represents a recovery key password for a host.
type HostRecoveryKeyPassword struct {
	Password  string
	UpdatedAt time.Time
}

// HostRecoveryLockAction represents a host that needs recovery lock action.
type HostRecoveryLockAction struct {
	HostID   uint
	HostUUID string
	TeamID   *uint
	Status   *fleet.MDMDeliveryStatus // nil means no existing record
}

// HostPendingRecoveryLock represents a host with a pending SetRecoveryLock command
// and the result status from nano_command_results.
type HostPendingRecoveryLock struct {
	HostID              uint
	HostUUID            string
	SetCommandUUID      string
	SetCommandStatus    string // "Acknowledged", "Error", "CommandFormatError", or empty if no result yet
	SetCommandErrorInfo string // Error details if status is Error
}

// HostStaleVerifyingRecoveryLock represents a host with status='verifying' where
// the VerifyRecoveryLock command has not been acknowledged (NotNow or no result).
type HostStaleVerifyingRecoveryLock struct {
	HostID   uint
	HostUUID string
}

// Datastore defines the data access interface for recovery key passwords.
type Datastore interface {
	// SetHostRecoveryKeyPassword generates a new recovery key password,
	// encrypts it, and stores it for the given host. Returns the plaintext password.
	SetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (string, error)

	// GetHostRecoveryKeyPassword retrieves and decrypts the recovery key password
	// for the given host.
	GetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (*HostRecoveryKeyPassword, error)

	// GetHostsForRecoveryLockAction returns hosts that need recovery lock password action:
	// - Teams with enable_recovery_lock_password = true
	// - macOS 11.5+, MDM enrolled
	// - No password OR status = failed
	GetHostsForRecoveryLockAction(ctx context.Context) ([]HostRecoveryLockAction, error)

	// SetRecoveryLockPending sets the recovery lock status to pending with the given set command UUID.
	// This is called when a SetRecoveryLock command is enqueued.
	SetRecoveryLockPending(ctx context.Context, hostID uint, setCommandUUID string) error

	// SetRecoveryLockVerifying marks the SetRecoveryLock command as acknowledged and updates
	// status to verifying with the given verify command UUID.
	SetRecoveryLockVerifying(ctx context.Context, hostID uint, verifyCommandUUID string) error

	// SetRecoveryLockVerified marks the recovery lock as verified (both commands succeeded).
	SetRecoveryLockVerified(ctx context.Context, hostID uint) error

	// SetRecoveryLockFailed marks the recovery lock as failed with the given error message.
	SetRecoveryLockFailed(ctx context.Context, hostID uint, errorMsg string) error

	// GetHostIDByVerifyCommandUUID returns the host ID associated with a VerifyRecoveryLock command UUID.
	GetHostIDByVerifyCommandUUID(ctx context.Context, verifyCommandUUID string) (uint, error)

	// GetPendingRecoveryLockHosts returns hosts with status='pending' along with
	// the SetRecoveryLock command result status from nano_command_results.
	// This is used by the cron job to check for acknowledged or failed Set commands.
	GetPendingRecoveryLockHosts(ctx context.Context) ([]HostPendingRecoveryLock, error)
}
