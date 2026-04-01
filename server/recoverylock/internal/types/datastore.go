// Package types defines internal types for the recovery lock bounded context.
// This package is internal and should not be imported from outside the recoverylock module.
package types

import (
	"context"
	"time"
)

// Datastore defines the internal database operations for recovery lock.
// This interface is NOT exported outside the recoverylock module.
type Datastore interface {
	PasswordDatastore
	StatusDatastore
	RotationDatastore
	CronDatastore
}

// PasswordDatastore handles password CRUD operations.
type PasswordDatastore interface {
	// SetHostsRecoveryLockPasswords sets recovery lock passwords for multiple hosts.
	// Uses INSERT ... ON DUPLICATE KEY UPDATE for upsert behavior.
	SetHostsRecoveryLockPasswords(ctx context.Context, passwords []PasswordPayload) error

	// GetHostRecoveryLockPassword retrieves the decrypted password for a host.
	GetHostRecoveryLockPassword(ctx context.Context, hostUUID string) (*Password, error)

	// DeleteHostRecoveryLockPassword soft-deletes the password record for a host.
	DeleteHostRecoveryLockPassword(ctx context.Context, hostUUID string) error

	// MarkRecoveryLockPasswordViewed marks a password as viewed and schedules auto-rotation.
	// Returns the scheduled auto-rotation time.
	MarkRecoveryLockPasswordViewed(ctx context.Context, hostUUID string, autoRotateDuration time.Duration) (*time.Time, error)
}

// StatusDatastore handles status operations.
type StatusDatastore interface {
	// GetHostRecoveryLockPasswordStatus returns the full status for a host.
	GetHostRecoveryLockPasswordStatus(ctx context.Context, hostUUID string) (*HostStatus, error)

	// SetRecoveryLockVerified marks a recovery lock as successfully verified.
	SetRecoveryLockVerified(ctx context.Context, hostUUID string) error

	// SetRecoveryLockFailed marks a recovery lock operation as failed.
	SetRecoveryLockFailed(ctx context.Context, hostUUID string, errorMsg string) error

	// ClearRecoveryLockPendingStatus clears pending status for hosts (for retry).
	ClearRecoveryLockPendingStatus(ctx context.Context, hostUUIDs []string) error

	// GetRecoveryLockOperationType returns the current operation type for a host.
	GetRecoveryLockOperationType(ctx context.Context, hostUUID string) (string, error)

	// ResetRecoveryLockForRetry resets a failed recovery lock for retry.
	ResetRecoveryLockForRetry(ctx context.Context, hostUUID string) error

	// GetHostsStatusBulk returns status for multiple hosts.
	GetHostsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]*HostStatus, error)

	// GetHostUUIDsByStatus returns host UUIDs with the given status.
	GetHostUUIDsByStatus(ctx context.Context, status string) ([]string, error)

	// FilterHostUUIDsByStatus filters candidate UUIDs to those with the given status.
	FilterHostUUIDsByStatus(ctx context.Context, status string, candidateUUIDs []string) ([]string, error)
}

// RotationDatastore handles password rotation operations.
type RotationDatastore interface {
	// InitiateRecoveryLockRotation starts a password rotation.
	InitiateRecoveryLockRotation(ctx context.Context, hostUUID string, newEncryptedPassword []byte) error

	// CompleteRecoveryLockRotation completes a successful rotation.
	CompleteRecoveryLockRotation(ctx context.Context, hostUUID string) error

	// FailRecoveryLockRotation marks a rotation as failed.
	FailRecoveryLockRotation(ctx context.Context, hostUUID string, errorMsg string) error

	// ClearRecoveryLockRotation clears a pending rotation.
	ClearRecoveryLockRotation(ctx context.Context, hostUUID string) error

	// GetRecoveryLockRotationStatus returns the rotation status for a host.
	GetRecoveryLockRotationStatus(ctx context.Context, hostUUID string) (*RotationStatus, error)

	// HasPendingRecoveryLockRotation checks if a rotation is in progress.
	HasPendingRecoveryLockRotation(ctx context.Context, hostUUID string) (bool, error)
}

// CronDatastore handles cron job database operations.
type CronDatastore interface {
	// GetHostsForRecoveryLockAction returns hosts that need recovery lock set.
	// These are ARM hosts with MDM enrollment and feature enabled but no password.
	GetHostsForRecoveryLockAction(ctx context.Context, eligibleUUIDs []string) ([]string, error)

	// RestoreRecoveryLockForReenabledHosts restores recovery lock for hosts
	// where the feature was disabled and then re-enabled.
	RestoreRecoveryLockForReenabledHosts(ctx context.Context) (int64, error)

	// ClaimHostsForRecoveryLockClear claims hosts for recovery lock clear operation.
	// Returns the UUIDs of claimed hosts.
	ClaimHostsForRecoveryLockClear(ctx context.Context) ([]string, error)

	// GetHostsForAutoRotation returns hosts with passwords scheduled for auto-rotation.
	GetHostsForAutoRotation(ctx context.Context) ([]HostAutoRotationInfo, error)
}

// PasswordPayload is the payload for setting a recovery lock password.
type PasswordPayload struct {
	// HostUUID is the host's UUID.
	HostUUID string
	// EncryptedPassword is the encrypted password.
	EncryptedPassword []byte
	// Status is the initial status (typically "pending").
	Status string
	// OperationType is the operation type ("install" or "remove").
	OperationType string
}

// Password represents a stored recovery lock password.
type Password struct {
	// Password is the plaintext (decrypted) password.
	Password string
	// UpdatedAt is when the password was last changed.
	UpdatedAt time.Time
	// AutoRotateAt is when the password is scheduled for auto-rotation.
	AutoRotateAt *time.Time
}

// HostStatus represents the full status of a host's recovery lock.
type HostStatus struct {
	// Status is the MDM delivery status.
	Status string
	// OperationType is "install" or "remove".
	OperationType string
	// ErrorMessage contains error details if failed.
	ErrorMessage string
	// PasswordAvailable indicates if a password is stored.
	PasswordAvailable bool
	// HasPendingRotation indicates if a rotation is in progress.
	HasPendingRotation bool
}

// RotationStatus represents the status of a password rotation.
type RotationStatus struct {
	// HasPendingRotation indicates if a rotation is in progress.
	HasPendingRotation bool
	// Status is the current status.
	Status string
	// ErrorMessage contains error details if failed.
	ErrorMessage string
}

// HostAutoRotationInfo contains info for a host scheduled for auto-rotation.
type HostAutoRotationInfo struct {
	// HostID is the host's numeric ID.
	HostID uint
	// HostUUID is the host's UUID.
	HostUUID string
	// HostDisplayName is the host's display name for activity logging.
	HostDisplayName string
}
