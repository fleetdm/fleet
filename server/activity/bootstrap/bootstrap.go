// Package bootstrap provides the public entry point for the activity bounded context.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
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
	userProvider activity.UserProvider,
	hostProvider activity.HostProvider,
	logger kitlog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(authorizer, ds, userProvider, hostProvider, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}
