// Package service implements the MDM settings status aggregator.
package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus"
	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus/api"
	rlapi "github.com/fleetdm/fleet/v4/server/recoverylock/api"
)

// Service implements the api.Service interface.
type Service struct {
	recoveryLock mdmsettingsstatus.RecoveryLockStatusProvider
	profiles     mdmsettingsstatus.ProfilesStatusProvider
	declarations mdmsettingsstatus.DeclarationsStatusProvider
	fileVault    mdmsettingsstatus.FileVaultStatusProvider
}

// New creates a new MDM settings status aggregator service.
func New(
	recoveryLock mdmsettingsstatus.RecoveryLockStatusProvider,
	profiles mdmsettingsstatus.ProfilesStatusProvider,
	declarations mdmsettingsstatus.DeclarationsStatusProvider,
	fileVault mdmsettingsstatus.FileVaultStatusProvider,
) *Service {
	return &Service{
		recoveryLock: recoveryLock,
		profiles:     profiles,
		declarations: declarations,
		fileVault:    fileVault,
	}
}

// GetHostsOverallStatus returns the computed overall MDM status for hosts.
func (s *Service) GetHostsOverallStatus(ctx context.Context, hostUUIDs []string) (map[string]mdmsettingsstatus.Status, error) {
	if len(hostUUIDs) == 0 {
		return make(map[string]mdmsettingsstatus.Status), nil
	}

	// Get status from all components
	rlStatus, err := s.recoveryLock.GetHostsStatusBulk(ctx, hostUUIDs)
	if err != nil {
		return nil, err
	}

	profStatus, err := s.profiles.GetProfilesStatusBulk(ctx, hostUUIDs)
	if err != nil {
		return nil, err
	}

	declStatus, err := s.declarations.GetDeclarationsStatusBulk(ctx, hostUUIDs)
	if err != nil {
		return nil, err
	}

	fvStatus, err := s.fileVault.GetFileVaultStatusBulk(ctx, hostUUIDs)
	if err != nil {
		return nil, err
	}

	// Aggregate using hierarchical logic
	result := make(map[string]mdmsettingsstatus.Status, len(hostUUIDs))
	for _, uuid := range hostUUIDs {
		result[uuid] = aggregateStatus(
			toStatus(rlStatus[uuid]),
			profStatus[uuid],
			declStatus[uuid],
			fvStatus[uuid],
		)
	}

	return result, nil
}

// FilterHostsByOverallStatus returns host UUIDs matching an overall status.
func (s *Service) FilterHostsByOverallStatus(ctx context.Context, targetStatus mdmsettingsstatus.Status, candidateUUIDs []string) ([]string, error) {
	// Get hosts matching the target status from each component
	rlMatches, err := s.recoveryLock.FilterHostsByStatus(ctx, toRLStatus(targetStatus), candidateUUIDs)
	if err != nil {
		return nil, err
	}

	profMatches, err := s.profiles.FilterHostsByProfileStatus(ctx, targetStatus, candidateUUIDs)
	if err != nil {
		return nil, err
	}

	declMatches, err := s.declarations.FilterHostsByDeclarationStatus(ctx, targetStatus, candidateUUIDs)
	if err != nil {
		return nil, err
	}

	fvMatches, err := s.fileVault.FilterHostsByFileVaultStatus(ctx, targetStatus, candidateUUIDs)
	if err != nil {
		return nil, err
	}

	// Combine based on hierarchical logic
	return combineFilters(ctx, s, targetStatus, rlMatches, profMatches, declMatches, fvMatches, candidateUUIDs)
}

// aggregateStatus computes the overall status using hierarchical logic:
// failed > pending > verifying > verified
func aggregateStatus(statuses ...mdmsettingsstatus.Status) mdmsettingsstatus.Status {
	hasPending := false
	hasVerifying := false
	allVerified := true

	for _, status := range statuses {
		switch status {
		case mdmsettingsstatus.StatusFailed:
			// If any component is failed, overall is failed
			return mdmsettingsstatus.StatusFailed
		case mdmsettingsstatus.StatusPending:
			hasPending = true
			allVerified = false
		case mdmsettingsstatus.StatusVerifying:
			hasVerifying = true
			allVerified = false
		case mdmsettingsstatus.StatusVerified:
			// Continue checking
		default:
			// Unknown status, treat as pending
			hasPending = true
			allVerified = false
		}
	}

	if hasPending {
		return mdmsettingsstatus.StatusPending
	}
	if hasVerifying {
		return mdmsettingsstatus.StatusVerifying
	}
	if allVerified && len(statuses) > 0 {
		return mdmsettingsstatus.StatusVerified
	}

	// Default to pending if no status
	return mdmsettingsstatus.StatusPending
}

// combineFilters combines filter results based on hierarchical logic.
func combineFilters(
	ctx context.Context,
	s *Service,
	targetStatus mdmsettingsstatus.Status,
	rl, prof, decl, fv []string,
	candidateUUIDs []string,
) ([]string, error) {
	switch targetStatus {
	case mdmsettingsstatus.StatusFailed:
		// ANY component failed → host is "failed"
		// UNION of all failed hosts
		return union(rl, prof, decl, fv), nil

	case mdmsettingsstatus.StatusPending:
		// ANY component pending AND no component failed → host is "pending"
		pending := union(rl, prof, decl, fv)
		// Get failed hosts to subtract
		failed, err := s.getFailedHosts(ctx, candidateUUIDs)
		if err != nil {
			return nil, err
		}
		return subtract(pending, failed), nil

	case mdmsettingsstatus.StatusVerifying:
		// ANY component verifying AND no failed/pending → host is "verifying"
		verifying := union(rl, prof, decl, fv)
		failedOrPending, err := s.getFailedOrPendingHosts(ctx, candidateUUIDs)
		if err != nil {
			return nil, err
		}
		return subtract(verifying, failedOrPending), nil

	case mdmsettingsstatus.StatusVerified:
		// ALL components must be verified → INTERSECTION
		return intersection(rl, prof, decl, fv), nil

	default:
		return nil, nil
	}
}

// getFailedHosts returns hosts with any component in failed status.
func (s *Service) getFailedHosts(ctx context.Context, candidateUUIDs []string) ([]string, error) {
	rl, err := s.recoveryLock.FilterHostsByStatus(ctx, rlapi.StatusFailed, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	prof, err := s.profiles.FilterHostsByProfileStatus(ctx, mdmsettingsstatus.StatusFailed, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	decl, err := s.declarations.FilterHostsByDeclarationStatus(ctx, mdmsettingsstatus.StatusFailed, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	fv, err := s.fileVault.FilterHostsByFileVaultStatus(ctx, mdmsettingsstatus.StatusFailed, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	return union(rl, prof, decl, fv), nil
}

// getFailedOrPendingHosts returns hosts with any component in failed or pending status.
func (s *Service) getFailedOrPendingHosts(ctx context.Context, candidateUUIDs []string) ([]string, error) {
	failed, err := s.getFailedHosts(ctx, candidateUUIDs)
	if err != nil {
		return nil, err
	}

	rl, err := s.recoveryLock.FilterHostsByStatus(ctx, rlapi.StatusPending, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	prof, err := s.profiles.FilterHostsByProfileStatus(ctx, mdmsettingsstatus.StatusPending, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	decl, err := s.declarations.FilterHostsByDeclarationStatus(ctx, mdmsettingsstatus.StatusPending, candidateUUIDs)
	if err != nil {
		return nil, err
	}
	fv, err := s.fileVault.FilterHostsByFileVaultStatus(ctx, mdmsettingsstatus.StatusPending, candidateUUIDs)
	if err != nil {
		return nil, err
	}

	pending := union(rl, prof, decl, fv)
	return union(failed, pending), nil
}

// toStatus converts a recovery lock HostStatus to an aggregator Status.
func toStatus(s *rlapi.HostStatus) mdmsettingsstatus.Status {
	if s == nil {
		return mdmsettingsstatus.StatusVerified // No recovery lock = verified
	}
	switch s.Status {
	case rlapi.StatusPending:
		return mdmsettingsstatus.StatusPending
	case rlapi.StatusFailed:
		return mdmsettingsstatus.StatusFailed
	case rlapi.StatusVerifying:
		return mdmsettingsstatus.StatusVerifying
	case rlapi.StatusVerified:
		return mdmsettingsstatus.StatusVerified
	default:
		return mdmsettingsstatus.StatusPending
	}
}

// toRLStatus converts an aggregator Status to a recovery lock Status.
func toRLStatus(s mdmsettingsstatus.Status) rlapi.Status {
	switch s {
	case mdmsettingsstatus.StatusPending:
		return rlapi.StatusPending
	case mdmsettingsstatus.StatusFailed:
		return rlapi.StatusFailed
	case mdmsettingsstatus.StatusVerifying:
		return rlapi.StatusVerifying
	case mdmsettingsstatus.StatusVerified:
		return rlapi.StatusVerified
	default:
		return rlapi.StatusPending
	}
}

// Verify interface compliance
var _ api.Service = (*Service)(nil)
