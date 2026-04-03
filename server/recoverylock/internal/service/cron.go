package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/recoverylock"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
)

// SendCommands processes pending recovery lock operations.
// This includes setting, clearing, and rotating passwords.
func (s *Service) SendCommands(ctx context.Context) error {
	// Step 1: Restore recovery lock for hosts where feature was re-enabled
	if _, err := s.ds.RestoreRecoveryLockForReenabledHosts(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "restore recovery lock for re-enabled hosts")
	}

	// Step 2: Process SET commands for hosts that need passwords
	if err := s.sendSetCommands(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "send set commands")
	}

	// Step 3: Process CLEAR commands for hosts where feature is disabled
	if err := s.sendClearCommands(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "send clear commands")
	}

	// Step 4: Process auto-rotation for viewed passwords
	if err := s.sendAutoRotationCommands(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "send auto rotation commands")
	}

	return nil
}

// sendSetCommands sends SET recovery lock commands to eligible hosts.
func (s *Service) sendSetCommands(ctx context.Context) error {
	// Get eligible hosts from provider (external query for ARM, MDM enrolled, feature enabled)
	eligibleUUIDs, err := s.providers.GetEligibleHostUUIDs(ctx, recoverylock.EligibilityFilter{
		FeatureEnabled: boolPtr(true),
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get eligible hosts for set")
	}

	if len(eligibleUUIDs) == 0 {
		return nil
	}

	// Filter to hosts that need a password (no record or NULL status)
	hostUUIDs, err := s.ds.GetHostsForRecoveryLockAction(ctx, eligibleUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	if len(hostUUIDs) == 0 {
		return nil
	}

	// Generate passwords for each host
	payloads := make([]types.PasswordPayload, 0, len(hostUUIDs))
	passwords := make(map[string]string, len(hostUUIDs))
	for _, uuid := range hostUUIDs {
		password := GeneratePassword()
		encrypted, err := s.encryptor.Encrypt(password)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypt password")
		}
		payloads = append(payloads, types.PasswordPayload{
			HostUUID:          uuid,
			EncryptedPassword: encrypted,
			Status:            "pending",
			OperationType:     "install",
		})
		passwords[uuid] = password
	}

	// Store passwords with pending status
	if err := s.ds.SetHostsRecoveryLockPasswords(ctx, payloads); err != nil {
		return ctxerr.Wrap(ctx, err, "set hosts recovery lock passwords")
	}

	// Send MDM commands
	cmdUUID := GenerateUUID()
	failedUUIDs := make([]string, 0)
	for uuid, password := range passwords {
		if err := s.providers.SetRecoveryLock(ctx, []string{uuid}, cmdUUID, password); err != nil {
			failedUUIDs = append(failedUUIDs, uuid)
		}
	}

	// Reset status for hosts that failed to enqueue
	if len(failedUUIDs) > 0 {
		if err := s.ds.ClearRecoveryLockPendingStatus(ctx, failedUUIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "clear pending status for failed hosts")
		}
	}

	return nil
}

// sendClearCommands sends CLEAR recovery lock commands.
func (s *Service) sendClearCommands(ctx context.Context) error {
	// Claim hosts for clear operation
	hostUUIDs, err := s.ds.ClaimHostsForRecoveryLockClear(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "claim hosts for clear")
	}

	if len(hostUUIDs) == 0 {
		return nil
	}

	// Get passwords and send clear commands
	failedUUIDs := make([]string, 0)
	for _, uuid := range hostUUIDs {
		password, err := s.ds.GetHostRecoveryLockPassword(ctx, uuid)
		if err != nil {
			failedUUIDs = append(failedUUIDs, uuid)
			continue
		}

		cmdUUID := GenerateUUID()
		if err := s.providers.ClearRecoveryLock(ctx, []string{uuid}, cmdUUID, password.Password); err != nil {
			failedUUIDs = append(failedUUIDs, uuid)
		}
	}

	// Reset status for hosts that failed to enqueue
	if len(failedUUIDs) > 0 {
		if err := s.ds.ClearRecoveryLockPendingStatus(ctx, failedUUIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "clear pending status for failed hosts")
		}
	}

	return nil
}

// sendAutoRotationCommands sends rotation commands for viewed passwords.
func (s *Service) sendAutoRotationCommands(ctx context.Context) error {
	// Get hosts due for auto-rotation
	hosts, err := s.ds.GetHostsForAutoRotation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for auto rotation")
	}

	if len(hosts) == 0 {
		return nil
	}

	// Process each host
	for _, host := range hosts {
		// Get current password
		currentPassword, err := s.ds.GetHostRecoveryLockPassword(ctx, host.HostUUID)
		if err != nil {
			continue // Skip this host
		}

		// Generate new password
		newPassword := GeneratePassword()
		encryptedPassword, err := s.encryptor.Encrypt(newPassword)
		if err != nil {
			continue // Skip this host
		}

		// Initiate rotation
		if err := s.ds.InitiateRecoveryLockRotation(ctx, host.HostUUID, encryptedPassword); err != nil {
			continue // Skip this host
		}

		// Send rotation command
		cmdUUID := GenerateUUID()
		if err := s.providers.RotateRecoveryLock(ctx, host.HostUUID, cmdUUID, currentPassword.Password, newPassword); err != nil {
			// Clear the rotation state
			_ = s.ds.ClearRecoveryLockRotation(ctx, host.HostUUID)
			continue // Skip this host
		}

		// Log activity if we have host info
		if host.HostID > 0 {
			_ = s.providers.LogRecoveryLockRotated(ctx, host.HostID, host.HostDisplayName)
		}
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
