package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/recoverylock/api"
)

// resultsHandler handles MDM command results for recovery lock operations.
type resultsHandler struct {
	ds interface {
		SetRecoveryLockVerified(ctx context.Context, hostUUID string) error
		SetRecoveryLockFailed(ctx context.Context, hostUUID string, errorMsg string) error
		DeleteHostRecoveryLockPassword(ctx context.Context, hostUUID string) error
		CompleteRecoveryLockRotation(ctx context.Context, hostUUID string) error
		FailRecoveryLockRotation(ctx context.Context, hostUUID string, errorMsg string) error
		GetRecoveryLockOperationType(ctx context.Context, hostUUID string) (string, error)
		HasPendingRecoveryLockRotation(ctx context.Context, hostUUID string) (bool, error)
		ResetRecoveryLockForRetry(ctx context.Context, hostUUID string) error
	}
}

// NewResultsHandler creates a handler for processing MDM command results.
func (s *Service) NewResultsHandler() api.MDMCommandResultsHandler {
	return &resultsHandler{ds: s.ds}
}

// Handle processes the result of an MDM command.
func (h *resultsHandler) Handle(ctx context.Context, result *api.CommandResult) error {
	if result == nil {
		return nil
	}

	// Only handle SetRecoveryLock commands
	if result.RequestType != "SetRecoveryLock" {
		return nil
	}

	hostUUID := result.HostUUID

	// Check if this is a rotation in progress
	hasPending, err := h.ds.HasPendingRecoveryLockRotation(ctx, hostUUID)
	if err != nil {
		return fmt.Errorf("check pending rotation: %w", err)
	}

	// Get the operation type
	opType, err := h.ds.GetRecoveryLockOperationType(ctx, hostUUID)
	if err != nil {
		return fmt.Errorf("get operation type: %w", err)
	}

	switch result.Status {
	case "Acknowledged":
		return h.handleAcknowledged(ctx, hostUUID, opType, hasPending)
	case "Error":
		return h.handleError(ctx, hostUUID, opType, hasPending, result.ErrorChain)
	default:
		// For other statuses (NotNow, etc.), just update to pending for retry
		return nil
	}
}

// handleAcknowledged processes a successful MDM command.
func (h *resultsHandler) handleAcknowledged(ctx context.Context, hostUUID, opType string, hasPending bool) error {
	switch {
	case hasPending:
		// Rotation completed successfully
		return h.ds.CompleteRecoveryLockRotation(ctx, hostUUID)
	case opType == "remove":
		// Clear completed successfully - delete the password record
		return h.ds.DeleteHostRecoveryLockPassword(ctx, hostUUID)
	default:
		// Set completed successfully
		return h.ds.SetRecoveryLockVerified(ctx, hostUUID)
	}
}

// handleError processes a failed MDM command.
func (h *resultsHandler) handleError(ctx context.Context, hostUUID, opType string, hasPending bool, errorChain []api.ErrorChainItem) error {
	errorMsg := formatErrorMessage(errorChain)

	// Check for specific error codes that require special handling
	isPasswordMismatch := hasErrorCode(errorChain, 1000) // MDMClientError.InvalidCommand for wrong password

	switch {
	case hasPending:
		// Rotation failed
		if isPasswordMismatch {
			// Password was changed externally - mark as failed and allow retry
			return h.ds.FailRecoveryLockRotation(ctx, hostUUID, errorMsg)
		}
		return h.ds.FailRecoveryLockRotation(ctx, hostUUID, errorMsg)
	case opType == "remove":
		// Clear failed
		if isPasswordMismatch {
			// Our stored password doesn't match - reset for manual intervention
			return h.ds.ResetRecoveryLockForRetry(ctx, hostUUID)
		}
		return h.ds.SetRecoveryLockFailed(ctx, hostUUID, errorMsg)
	default:
		// Set failed
		return h.ds.SetRecoveryLockFailed(ctx, hostUUID, errorMsg)
	}
}

// formatErrorMessage creates a human-readable error message from the error chain.
func formatErrorMessage(errorChain []api.ErrorChainItem) string {
	if len(errorChain) == 0 {
		return "unknown error"
	}

	// Use the first error with a description
	for _, e := range errorChain {
		if e.USEnglishDescription != "" {
			return e.USEnglishDescription
		}
		if e.LocalizedDescription != "" {
			return e.LocalizedDescription
		}
	}

	// Fallback to error code
	return fmt.Sprintf("error code: %d", errorChain[0].ErrorCode)
}

// hasErrorCode checks if the error chain contains a specific error code.
func hasErrorCode(errorChain []api.ErrorChainItem, code int) bool {
	for _, e := range errorChain {
		if e.ErrorCode == code {
			return true
		}
	}
	return false
}
