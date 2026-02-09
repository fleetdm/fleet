// Package bootstrap provides the public entry point for the activity bounded context.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/log"
)

// New creates a new activity bounded context and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	authorizer platform_authz.Authorizer,
	providers activity.DataProviders,
	logger kitlog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(authorizer, ds, providers, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}

// NewForUnitTests creates an activity NewActivityService backed by a noop store
// (no database required). This is useful for unit tests that need webhook behavior
// without a real database connection.
func NewForUnitTests(
	providers activity.DataProviders,
	logger kitlog.Logger,
) api.NewActivityService {
	return service.NewService(&noopAuthorizer{}, &noopStore{}, providers, logger)
}

// noopAuthorizer allows all actions (appropriate for unit tests).
type noopAuthorizer struct{}

func (a *noopAuthorizer) Authorize(_ context.Context, _ platform_authz.AuthzTyper, _ platform_authz.Action) error {
	return nil
}

// noopStore is a datastore that does nothing (appropriate for unit tests that
// only need webhook behavior).
type noopStore struct{}

func (s *noopStore) ListActivities(_ context.Context, _ types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (s *noopStore) ListHostPastActivities(_ context.Context, _ uint, _ types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (s *noopStore) MarkActivitiesAsStreamed(_ context.Context, _ []uint) error {
	return nil
}

func (s *noopStore) NewActivity(_ context.Context, _ *api.User, _ api.ActivityDetails, _ []byte, _ time.Time) error {
	return nil
}
