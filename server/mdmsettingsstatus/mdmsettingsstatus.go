// Package mdmsettingsstatus provides an aggregator service that combines
// MDM settings status from multiple sources (profiles, declarations, FileVault,
// and recovery lock) into a single overall status.
//
// This service is used by host listing endpoints to compute the overall MDM
// settings status displayed in the UI.
package mdmsettingsstatus

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/recoverylock/api"
)

// Status represents the overall MDM settings status for a host.
type Status string

const (
	// StatusPending indicates at least one component has pending changes.
	StatusPending Status = "pending"
	// StatusFailed indicates at least one component has a failed status.
	StatusFailed Status = "failed"
	// StatusVerifying indicates at least one component is being verified.
	StatusVerifying Status = "verifying"
	// StatusVerified indicates all components are verified.
	StatusVerified Status = "verified"
)

// IsValid returns true if the status is a known valid status.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusFailed, StatusVerifying, StatusVerified:
		return true
	default:
		return false
	}
}

// RecoveryLockStatusProvider provides recovery lock status information.
// This is implemented by the recoverylock bounded context.
type RecoveryLockStatusProvider interface {
	// GetHostsStatusBulk returns recovery lock status for multiple hosts.
	GetHostsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]*api.HostStatus, error)
	// FilterHostsByStatus returns host UUIDs that match the given status.
	FilterHostsByStatus(ctx context.Context, status api.Status, candidateUUIDs []string) ([]string, error)
}

// ProfilesStatusProvider provides profiles status information.
// This is implemented by the legacy datastore.
type ProfilesStatusProvider interface {
	// GetProfilesStatusBulk returns profiles status for multiple hosts.
	GetProfilesStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]Status, error)
	// FilterHostsByProfileStatus returns host UUIDs matching the given status.
	FilterHostsByProfileStatus(ctx context.Context, status Status, candidateUUIDs []string) ([]string, error)
}

// DeclarationsStatusProvider provides declarations status information.
// This is implemented by the legacy datastore.
type DeclarationsStatusProvider interface {
	// GetDeclarationsStatusBulk returns declarations status for multiple hosts.
	GetDeclarationsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]Status, error)
	// FilterHostsByDeclarationStatus returns host UUIDs matching the given status.
	FilterHostsByDeclarationStatus(ctx context.Context, status Status, candidateUUIDs []string) ([]string, error)
}

// FileVaultStatusProvider provides FileVault status information.
// This is implemented by the legacy datastore.
type FileVaultStatusProvider interface {
	// GetFileVaultStatusBulk returns FileVault status for multiple hosts.
	GetFileVaultStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]Status, error)
	// FilterHostsByFileVaultStatus returns host UUIDs matching the given status.
	FilterHostsByFileVaultStatus(ctx context.Context, status Status, candidateUUIDs []string) ([]string, error)
}
