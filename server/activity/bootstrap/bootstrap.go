// Package bootstrap provides the public entry point for the activity bounded context.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

// AuthMiddleware is a type alias for endpoint middleware functions.
// This is the type expected for authentication middleware when registering routes.
type AuthMiddleware = func(endpoint.Endpoint) endpoint.Endpoint

// New creates a new activity bounded context and returns its service and route handler.
//
// Parameters:
//   - primary: primary MySQL database connection for writes
//   - replica: replica MySQL database connection for reads
//   - authorizer: authorization checker (injected from serve.go)
//   - userProvider: ACL adapter for fetching user data from legacy service
//   - logger: logger for the service
//
// Returns:
//   - api.Service: the public activity service interface for external consumers
//   - func(AuthMiddleware) eu.HandlerRoutesFunc: function to create routes with auth middleware
func New(
	primary, replica *sqlx.DB,
	authorizer platform_authz.Authorizer,
	userProvider activity.UserProvider,
	logger kitlog.Logger,
) (api.Service, func(authMiddleware AuthMiddleware) eu.HandlerRoutesFunc) {
	// Create the datastore
	ds := mysql.NewDatastore(primary, replica)

	// Create the service (implements api.Service)
	svc := service.NewService(authorizer, ds, userProvider, logger)

	// Return the service and a function that creates route handlers with auth middleware
	routesFn := func(authMiddleware AuthMiddleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}
