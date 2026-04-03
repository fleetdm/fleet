// Package recoverylockacl provides an Anti-Corruption Layer (ACL) adapter
// that implements the recoverylock.DataProviders interface using legacy Fleet services.
//
// This adapter bridges the gap between the recovery lock bounded context and
// the existing Fleet codebase, allowing the bounded context to remain isolated
// while still integrating with the rest of the system.
package recoverylockacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/recoverylock"
)

// FleetAdapter implements recoverylock.DataProviders using Fleet services.
type FleetAdapter struct {
	ds        fleet.Datastore
	svc       fleet.Service
	commander recoverylock.CommanderProvider
}

// New creates a new FleetAdapter.
func New(ds fleet.Datastore, svc fleet.Service, commander recoverylock.CommanderProvider) *FleetAdapter {
	return &FleetAdapter{
		ds:        ds,
		svc:       svc,
		commander: commander,
	}
}

// GetHostLite returns minimal host information by host ID.
func (a *FleetAdapter) GetHostLite(ctx context.Context, hostID uint) (*recoverylock.Host, error) {
	host, err := a.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return toRecoveryLockHost(host), nil
}

// GetHostByUUID returns minimal host information by host UUID.
func (a *FleetAdapter) GetHostByUUID(ctx context.Context, uuid string) (*recoverylock.Host, error) {
	host, err := a.ds.HostByIdentifier(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return toRecoveryLockHost(host), nil
}

// GetEligibleHostUUIDs returns UUIDs of hosts eligible for recovery lock operations.
func (a *FleetAdapter) GetEligibleHostUUIDs(ctx context.Context, filter recoverylock.EligibilityFilter) ([]string, error) {
	// For now, delegate to the legacy datastore method
	// In a full implementation, this would use the filter parameters
	// to query hosts that meet the eligibility criteria
	return a.ds.GetHostsForRecoveryLockAction(ctx)
}

// IsRecoveryLockEnabled returns whether recovery lock is enabled for a team.
func (a *FleetAdapter) IsRecoveryLockEnabled(ctx context.Context, teamID *uint) (bool, error) {
	if teamID != nil {
		team, err := a.ds.TeamWithExtras(ctx, *teamID)
		if err != nil {
			return false, err
		}
		return team.Config.MDM.EnableRecoveryLockPassword, nil
	}

	// Check global config
	config, err := a.ds.AppConfig(ctx)
	if err != nil {
		return false, err
	}
	return config.MDM.EnableRecoveryLockPassword.Value, nil
}

// GetAutoRotationDuration returns the duration after which viewed passwords
// should be automatically rotated. Returns 0 if auto-rotation is disabled.
func (a *FleetAdapter) GetAutoRotationDuration(ctx context.Context) (int, error) {
	// Auto-rotation is currently hardcoded to 1 hour (3600 seconds)
	// In a full implementation, this could be configurable
	return 3600, nil
}

// SetRecoveryLock sends a SetRecoveryLock MDM command to the specified hosts.
func (a *FleetAdapter) SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string, password string) error {
	return a.commander.SetRecoveryLock(ctx, hostUUIDs, cmdUUID, password)
}

// ClearRecoveryLock sends a ClearRecoveryLock MDM command to the specified hosts.
func (a *FleetAdapter) ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string, password string) error {
	return a.commander.ClearRecoveryLock(ctx, hostUUIDs, cmdUUID, password)
}

// RotateRecoveryLock sends commands to clear and set a new recovery lock password.
func (a *FleetAdapter) RotateRecoveryLock(ctx context.Context, hostUUID string, cmdUUID string, oldPassword string, newPassword string) error {
	return a.commander.RotateRecoveryLock(ctx, hostUUID, cmdUUID, oldPassword, newPassword)
}

// LogRecoveryLockViewed logs that a user viewed a recovery lock password.
func (a *FleetAdapter) LogRecoveryLockViewed(ctx context.Context, hostID uint, hostDisplayName string) error {
	// Activity logging would go here
	// For now, this is a no-op as activity logging requires the service layer
	return nil
}

// LogRecoveryLockRotated logs that a recovery lock password was rotated.
func (a *FleetAdapter) LogRecoveryLockRotated(ctx context.Context, hostID uint, hostDisplayName string) error {
	// Activity logging would go here
	// For now, this is a no-op as activity logging requires the service layer
	return nil
}

// toRecoveryLockHost converts a Fleet host to a recoverylock.Host.
func toRecoveryLockHost(host *fleet.Host) *recoverylock.Host {
	if host == nil {
		return nil
	}
	return &recoverylock.Host{
		ID:            host.ID,
		UUID:          host.UUID,
		TeamID:        host.TeamID,
		Platform:      host.Platform,
		HardwareModel: host.HardwareModel,
	}
}

// Verify interface compliance
var _ recoverylock.DataProviders = (*FleetAdapter)(nil)
