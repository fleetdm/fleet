// Package bootstrap is where serve.go builds the notifications bounded
// context.
package bootstrap

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/notifications"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/service"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/go-kit/kit/endpoint"
)

// New returns the context's service, and a function that takes the auth
// middleware for its routes. server/service owns device authentication, so
// the middleware can only be supplied from outside.
func New(
	dbConns *platform_mysql.DBConnections,
	providers notifications.DataProviders,
	logger *slog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(ds, providers, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}

// RegisterTracingTiers classifies this context's routes for trace sampling.
func RegisterTracingTiers(registry *tracing.Registry) {
	service.RegisterTracingTiers(registry)
}
