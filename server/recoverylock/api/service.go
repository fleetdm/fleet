// Package api defines the public service interfaces for the recovery lock bounded context.
// External packages should only import from this package, never from internal/.
package api

import (
	"context"
	"time"
)

// Service is the composite interface for all recovery lock operations.
// Use bootstrap.New() to create an implementation.
type Service interface {
	PasswordService
	StatusService
	CronService
	ResultHandlerService
}

// PasswordService handles recovery lock password operations.
type PasswordService interface {
	// GetHostRecoveryLockPassword retrieves the recovery lock password for a host.
	// This marks the password as viewed and schedules auto-rotation if enabled.
	GetHostRecoveryLockPassword(ctx context.Context, hostID uint) (*Password, error)

	// RotateHostRecoveryLockPassword initiates password rotation for a host.
	// This generates a new password and sends MDM commands to apply it.
	RotateHostRecoveryLockPassword(ctx context.Context, hostID uint) error
}

// StatusService handles recovery lock status queries.
type StatusService interface {
	// GetHostRecoveryLockStatus returns the current recovery lock status for a host.
	GetHostRecoveryLockStatus(ctx context.Context, hostUUID string) (*HostStatus, error)

	// GetHostsStatusBulk returns recovery lock status for multiple hosts.
	// This is optimized for host listing enrichment.
	GetHostsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]*HostStatus, error)

	// FilterHostsByStatus returns host UUIDs that match the given status.
	// If candidateUUIDs is nil, considers all hosts.
	FilterHostsByStatus(ctx context.Context, status Status, candidateUUIDs []string) ([]string, error)

	// GetHostUUIDsByStatus returns all host UUIDs with the given status.
	GetHostUUIDsByStatus(ctx context.Context, status Status) ([]string, error)
}

// CronService handles scheduled recovery lock operations.
type CronService interface {
	// SendCommands processes pending recovery lock operations.
	// This includes setting, clearing, and rotating passwords.
	SendCommands(ctx context.Context) error
}

// ResultHandlerService handles MDM command results.
type ResultHandlerService interface {
	// NewResultsHandler creates a handler for processing MDM command results.
	NewResultsHandler() MDMCommandResultsHandler
}

// MDMCommandResultsHandler processes MDM command results for recovery lock commands.
type MDMCommandResultsHandler interface {
	// Handle processes the result of an MDM command.
	Handle(ctx context.Context, result *CommandResult) error
}

// Password represents a recovery lock password with metadata.
type Password struct {
	// Password is the plaintext recovery lock password.
	Password string
	// UpdatedAt is when the password was last changed.
	UpdatedAt time.Time
	// AutoRotateAt is when the password is scheduled for auto-rotation.
	// Nil if not scheduled.
	AutoRotateAt *time.Time
}

// HostStatus represents the current recovery lock status for a host.
type HostStatus struct {
	// Status is the current delivery status.
	Status Status
	// OperationType indicates whether this is an install or remove operation.
	OperationType OperationType
	// ErrorMessage contains the error if status is Failed.
	ErrorMessage string
	// PasswordAvailable indicates whether a password is stored for this host.
	PasswordAvailable bool
	// HasPendingRotation indicates whether a rotation is in progress.
	HasPendingRotation bool
}

// Status represents the MDM delivery status for a recovery lock operation.
type Status string

const (
	// StatusPending indicates the MDM command is queued but not yet acknowledged.
	StatusPending Status = "pending"
	// StatusVerifying indicates the command was sent and we're waiting for verification.
	StatusVerifying Status = "verifying"
	// StatusVerified indicates the command was successfully applied.
	StatusVerified Status = "verified"
	// StatusFailed indicates the command failed.
	StatusFailed Status = "failed"
)

// IsValid returns true if the status is a known valid status.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusVerifying, StatusVerified, StatusFailed:
		return true
	default:
		return false
	}
}

// OperationType represents the type of recovery lock operation.
type OperationType string

const (
	// OperationInstall indicates setting/installing a recovery lock password.
	OperationInstall OperationType = "install"
	// OperationRemove indicates clearing/removing a recovery lock password.
	OperationRemove OperationType = "remove"
)

// CommandResult represents the result of an MDM command.
type CommandResult struct {
	// CommandUUID is the unique identifier for the command.
	CommandUUID string
	// HostUUID is the UUID of the host that executed the command.
	HostUUID string
	// RequestType is the MDM command type (e.g., "SetRecoveryLock").
	RequestType string
	// Status is the result status (e.g., "Acknowledged", "Error").
	Status string
	// ErrorChain contains error details if the command failed.
	ErrorChain []ErrorChainItem
}

// ErrorChainItem represents an error in the MDM error chain.
type ErrorChainItem struct {
	ErrorCode            int
	ErrorDomain          string
	LocalizedDescription string
	USEnglishDescription string
}

// RotationStatus represents the status of a password rotation operation.
type RotationStatus struct {
	// HasPendingRotation indicates whether a rotation is in progress.
	HasPendingRotation bool
	// Status is the current status of the rotation.
	Status Status
	// ErrorMessage contains the error if the rotation failed.
	ErrorMessage string
}
