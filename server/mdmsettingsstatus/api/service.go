// Package api defines the public service interface for the MDM settings status aggregator.
package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdmsettingsstatus"
)

// Service is the interface for the MDM settings status aggregator.
type Service interface {
	// GetHostsOverallStatus returns the computed overall MDM status for hosts.
	// This combines status from all 4 components (profiles, declarations, FileVault, recovery lock)
	// using hierarchical logic: failed > pending > verifying > verified.
	GetHostsOverallStatus(ctx context.Context, hostUUIDs []string) (map[string]mdmsettingsstatus.Status, error)

	// FilterHostsByOverallStatus returns host UUIDs matching an overall status.
	// This is used for filtering before the main host query.
	// If candidateUUIDs is nil, considers all hosts.
	FilterHostsByOverallStatus(ctx context.Context, status mdmsettingsstatus.Status, candidateUUIDs []string) ([]string, error)
}
