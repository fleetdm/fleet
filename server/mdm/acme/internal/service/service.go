// Package service provides the service implementation for the ACME bounded context.
package service

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz  platform_authz.Authorizer
	store  types.Datastore
	logger *slog.Logger
}

// NewService creates a new activity service.
func NewService(
	authz platform_authz.Authorizer,
	store types.Datastore,
	logger *slog.Logger,
) *Service {
	return &Service{
		authz:  authz,
		store:  store,
		logger: logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

func (s *Service) NewNonce(ctx context.Context, identifier string) (string, error) {
	panic("unimplemented")
}

func (s *Service) GetDirectory(ctx context.Context, identifier string) (*types.Directory, error) {
	panic("unimplemented")
}
