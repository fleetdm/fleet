package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/recoverylock/api"
)

// DefaultAutoRotationDuration is the default duration after which viewed passwords
// are automatically rotated (1 hour).
const DefaultAutoRotationDuration = time.Hour

// GetHostRecoveryLockPassword retrieves the recovery lock password for a host.
func (s *Service) GetHostRecoveryLockPassword(ctx context.Context, hostID uint) (*api.Password, error) {
	// Get host info to get UUID
	host, err := s.providers.GetHostLite(ctx, hostID)
	if err != nil {
		return nil, err
	}

	// Get the password from the datastore
	password, err := s.ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	if err != nil {
		return nil, err
	}

	// Get auto-rotation duration from config
	autoRotateDuration, err := s.providers.GetAutoRotationDuration(ctx)
	if err != nil {
		return nil, err
	}

	// Mark as viewed and schedule auto-rotation if enabled
	var autoRotateAt *time.Time
	if autoRotateDuration > 0 {
		autoRotateAt, err = s.ds.MarkRecoveryLockPasswordViewed(ctx, host.UUID, time.Duration(autoRotateDuration)*time.Second)
		if err != nil {
			return nil, err
		}
	}

	// Log activity - ignore errors as activity logging is secondary
	displayName := host.UUID // Use UUID as fallback display name
	_ = s.providers.LogRecoveryLockViewed(ctx, hostID, displayName)

	return &api.Password{
		Password:     password.Password,
		UpdatedAt:    password.UpdatedAt,
		AutoRotateAt: autoRotateAt,
	}, nil
}

// RotateHostRecoveryLockPassword initiates password rotation for a host.
func (s *Service) RotateHostRecoveryLockPassword(ctx context.Context, hostID uint) error {
	// Get host info
	host, err := s.providers.GetHostLite(ctx, hostID)
	if err != nil {
		return err
	}

	// Check if rotation is already pending
	hasPending, err := s.ds.HasPendingRecoveryLockRotation(ctx, host.UUID)
	if err != nil {
		return err
	}
	if hasPending {
		// Rotation already in progress, nothing to do
		return nil
	}

	// Get the current password to use for clearing the old lock
	currentPassword, err := s.ds.GetHostRecoveryLockPassword(ctx, host.UUID)
	if err != nil {
		return err
	}

	// Generate a new password
	newPassword := GeneratePassword()

	// Encrypt the new password
	encryptedPassword, err := s.encryptor.Encrypt(newPassword)
	if err != nil {
		return err
	}

	// Initiate the rotation in the database
	if err := s.ds.InitiateRecoveryLockRotation(ctx, host.UUID, encryptedPassword); err != nil {
		return err
	}

	// Send the rotation command through the commander
	cmdUUID := GenerateUUID()
	if err := s.providers.RotateRecoveryLock(ctx, host.UUID, cmdUUID, currentPassword.Password, newPassword); err != nil {
		// Clear the rotation state since command failed
		_ = s.ds.ClearRecoveryLockRotation(ctx, host.UUID)
		return err
	}

	// Log activity - ignore errors as activity logging is secondary
	displayName := host.UUID
	_ = s.providers.LogRecoveryLockRotated(ctx, hostID, displayName)

	return nil
}
