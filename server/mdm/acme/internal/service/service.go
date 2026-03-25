// Package service provides the service implementation for the ACME bounded context.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
)

// Service is the activity bounded context service implementation.
type Service struct {
	store     types.Datastore
	nonces    *redis_nonces_store.RedisNoncesStore
	providers acme.DataProviders
	logger    *slog.Logger
}

// NewService creates a new activity service.
func NewService(
	store types.Datastore,
	redisPool fleet.RedisPool,
	providers acme.DataProviders,
	logger *slog.Logger,
) *Service {
	noncesStore := redis_nonces_store.New(redisPool)
	return &Service{
		store:     store,
		nonces:    noncesStore,
		providers: providers,
		logger:    logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

func (s *Service) NoncesStore() *redis_nonces_store.RedisNoncesStore {
	return s.nonces
}

// TODO(mna): I'm assuming we'll need this at some point (when we need to resolve only 1 URL), but not used for now.
// func (s *Service) getACMEURL(ctx context.Context, pathIdentifier string, suffixes ...string) (string, error) {
// 	appConfig, err := s.providers.AppConfig(ctx)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return s.getACMEURLWithBaseURL(ctx, appConfig.MDMUrl(), pathIdentifier, suffixes...)
// }

func (s *Service) getACMEURLWithBaseURL(_ context.Context, baseURL, pathIdentifier string, suffixes ...string) (string, error) {
	return commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/%s", pathIdentifier, strings.Join(suffixes, "/")), true)
}
