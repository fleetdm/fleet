// Package service provides the service implementation for the ACME bounded context.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz     platform_authz.Authorizer
	store     types.Datastore
	providers acme.DataProviders
	logger    *slog.Logger
}

// NewService creates a new activity service.
func NewService(
	authz platform_authz.Authorizer,
	store types.Datastore,
	providers acme.DataProviders,
	logger *slog.Logger,
) *Service {
	return &Service{
		authz:     authz,
		store:     store,
		providers: providers,
		logger:    logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

func (s *Service) NewNonce(ctx context.Context, identifier string) (string, error) {
	panic("unimplemented")
}

func (s *Service) GetDirectory(ctx context.Context, identifier string) (*types.Directory, error) {
	// authentication is via the identifier, that must exist as a valid ACME enrollment
	enrollment, err := s.store.GetACMEEnrollment(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if !enrollment.IsValid() {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("ACME enrollment").WithName(identifier))
	}

	appConfig, err := s.providers.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := types.AppleACMEBaseURL(appConfig.ServerSettings.ServerURL)
	newNonce, err := commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/new_nonce", identifier), true)
	if err != nil {
		return nil, err
	}
	newAccount, err := commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/new_account", identifier), true)
	if err != nil {
		return nil, err
	}
	newOrder, err := commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/new_order", identifier), true)
	if err != nil {
		return nil, err
	}

	return &types.Directory{
		NewNonce:   newNonce,
		NewAccount: newAccount,
		NewOrder:   newOrder,
	}, nil
}
