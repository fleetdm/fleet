// Package recoverylock implements the Recovery Lock bounded context for managing
// macOS recovery lock passwords on Apple Silicon devices.
//
// This module follows the bounded context pattern with exclusive ownership of the
// host_recovery_key_passwords table. All access to recovery lock data must go
// through the public API defined in the api/ subpackage.
//
// See README.md for architecture details and usage examples.
package recoverylock

import (
	"context"
)

// Host represents the minimal host information needed by the recovery lock module.
type Host struct {
	ID            uint
	UUID          string
	TeamID        *uint
	Platform      string
	HardwareModel string
}

// EligibilityFilter specifies criteria for finding hosts eligible for recovery lock operations.
type EligibilityFilter struct {
	// FeatureEnabled filters to hosts where the recovery lock feature is enabled.
	FeatureEnabled *bool
	// TeamIDs filters to hosts in specific teams. If nil, considers all teams.
	TeamIDs []uint
	// ExcludeHostUUIDs excludes specific hosts from the results.
	ExcludeHostUUIDs []string
}

// DataProviders defines the external dependencies required by the recovery lock module.
// These interfaces are implemented by ACL adapters that bridge to the legacy Fleet services.
type DataProviders interface {
	HostProvider
	ConfigProvider
	CommanderProvider
	ActivityProvider
}

// HostProvider provides access to host information.
type HostProvider interface {
	// GetHostLite returns minimal host information by host ID.
	GetHostLite(ctx context.Context, hostID uint) (*Host, error)
	// GetHostByUUID returns minimal host information by host UUID.
	GetHostByUUID(ctx context.Context, uuid string) (*Host, error)
	// GetEligibleHostUUIDs returns UUIDs of hosts eligible for recovery lock operations.
	// Eligible hosts are: macOS ARM, MDM enrolled, and have the feature enabled.
	GetEligibleHostUUIDs(ctx context.Context, filter EligibilityFilter) ([]string, error)
}

// ConfigProvider provides access to Fleet configuration.
type ConfigProvider interface {
	// IsRecoveryLockEnabled returns whether recovery lock is enabled for a team.
	// Pass nil for no-team (global) configuration.
	IsRecoveryLockEnabled(ctx context.Context, teamID *uint) (bool, error)
	// GetAutoRotationDuration returns the duration after which viewed passwords
	// should be automatically rotated. Returns 0 if auto-rotation is disabled.
	GetAutoRotationDuration(ctx context.Context) (int, error)
}

// CommanderProvider sends MDM commands to hosts.
type CommanderProvider interface {
	// SetRecoveryLock sends a SetRecoveryLock MDM command to the specified hosts.
	SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string, password string) error
	// ClearRecoveryLock sends a ClearRecoveryLock MDM command to the specified hosts.
	ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string, password string) error
	// RotateRecoveryLock sends commands to clear and set a new recovery lock password.
	RotateRecoveryLock(ctx context.Context, hostUUID string, cmdUUID string, oldPassword string, newPassword string) error
}

// ActivityProvider logs user activities.
type ActivityProvider interface {
	// LogRecoveryLockViewed logs that a user viewed a recovery lock password.
	LogRecoveryLockViewed(ctx context.Context, hostID uint, hostDisplayName string) error
	// LogRecoveryLockRotated logs that a recovery lock password was rotated.
	LogRecoveryLockRotated(ctx context.Context, hostID uint, hostDisplayName string) error
}

// PasswordEncryptor handles encryption/decryption of recovery lock passwords.
type PasswordEncryptor interface {
	// Encrypt encrypts a plaintext password.
	Encrypt(plaintext string) ([]byte, error)
	// Decrypt decrypts an encrypted password.
	Decrypt(ciphertext []byte) (string, error)
}
