// Package service provides the service implementation for the ACME bounded context.
package service

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz     platform_authz.Authorizer
	store     types.Datastore
	nonces    *redis_nonces_store.RedisNoncesStore
	providers acme.DataProviders
	logger    *slog.Logger
}

// NewService creates a new activity service.
func NewService(
	authz platform_authz.Authorizer,
	store types.Datastore,
	redisPool fleet.RedisPool,
	providers acme.DataProviders,
	logger *slog.Logger,
) *Service {
	noncesStore := redis_nonces_store.New(redisPool)
	return &Service{
		authz:     authz,
		store:     store,
		nonces:    noncesStore,
		providers: providers,
		logger:    logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)
