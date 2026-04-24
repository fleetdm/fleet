// Package bootstrap provides the public entry point for the activity bounded context.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-kit/kit/endpoint"
)

// New creates a new activity bounded context and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	authorizer platform_authz.Authorizer,
	providers activity.DataProviders,
	logger *slog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(authorizer, ds, providers, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}

// StubRoutes returns a HandlerRoutesFunc that registers activity HTTP routes
// using a no-op stub service. Intended for tests that need the routes present
// for apiendpoints.Init validation but do not exercise activity endpoints.
func StubRoutes(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
	return service.GetRoutes(&stubActivityService{}, authMiddleware)
}

type stubActivityService struct{}

func (*stubActivityService) ListActivities(_ context.Context, _ api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (*stubActivityService) ListHostPastActivities(_ context.Context, _ uint, _ api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (*stubActivityService) StreamActivities(_ context.Context, _ api.JSONLogger) error {
	return nil
}

func (*stubActivityService) NewActivity(_ context.Context, _ *api.User, _ api.ActivityDetails) error {
	return nil
}

func (*stubActivityService) CleanupExpiredActivities(_ context.Context, _ int, _ int) error {
	return nil
}

func (*stubActivityService) CleanupHostActivities(_ context.Context, _ []uint) error {
	return nil
}

// Ensure stubActivityService satisfies the api.Service interface at compile time.
var _ api.Service = (*stubActivityService)(nil)
