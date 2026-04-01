// Package service implements the recovery lock business logic.
// This package should only be imported within the recoverylock module.
package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/recoverylock"
	"github.com/fleetdm/fleet/v4/server/recoverylock/api"
	"github.com/fleetdm/fleet/v4/server/recoverylock/internal/types"
)

// Service implements the api.Service interface.
type Service struct {
	ds        types.Datastore
	providers recoverylock.DataProviders
	encryptor recoverylock.PasswordEncryptor
}

// New creates a new recovery lock service.
func New(ds types.Datastore, providers recoverylock.DataProviders, encryptor recoverylock.PasswordEncryptor) *Service {
	return &Service{
		ds:        ds,
		providers: providers,
		encryptor: encryptor,
	}
}

// GetHostRecoveryLockStatus returns the current recovery lock status for a host.
func (s *Service) GetHostRecoveryLockStatus(ctx context.Context, hostUUID string) (*api.HostStatus, error) {
	status, err := s.ds.GetHostRecoveryLockPasswordStatus(ctx, hostUUID)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, nil
	}
	return toAPIHostStatus(status), nil
}

// GetHostsStatusBulk returns recovery lock status for multiple hosts.
func (s *Service) GetHostsStatusBulk(ctx context.Context, hostUUIDs []string) (map[string]*api.HostStatus, error) {
	statusMap, err := s.ds.GetHostsStatusBulk(ctx, hostUUIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*api.HostStatus, len(statusMap))
	for uuid, status := range statusMap {
		result[uuid] = toAPIHostStatus(status)
	}
	return result, nil
}

// FilterHostsByStatus returns host UUIDs that match the given status.
func (s *Service) FilterHostsByStatus(ctx context.Context, status api.Status, candidateUUIDs []string) ([]string, error) {
	return s.ds.FilterHostUUIDsByStatus(ctx, string(status), candidateUUIDs)
}

// GetHostUUIDsByStatus returns all host UUIDs with the given status.
func (s *Service) GetHostUUIDsByStatus(ctx context.Context, status api.Status) ([]string, error) {
	return s.ds.GetHostUUIDsByStatus(ctx, string(status))
}

// toAPIHostStatus converts internal status to API status.
func toAPIHostStatus(status *types.HostStatus) *api.HostStatus {
	if status == nil {
		return nil
	}
	return &api.HostStatus{
		Status:             api.Status(status.Status),
		OperationType:      api.OperationType(status.OperationType),
		ErrorMessage:       status.ErrorMessage,
		PasswordAvailable:  status.PasswordAvailable,
		HasPendingRotation: status.HasPendingRotation,
	}
}

// Verify interface compliance
var _ api.Service = (*Service)(nil)
