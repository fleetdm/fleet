// Package bootstrap provides the public entry point for the ACME service modiule.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

// New creates a new ACME service modiule and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	redisPool fleet.RedisPool,
	providers acme.DataProviders,
	logger *slog.Logger,
) (api.Service, func() eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(ds, redisPool, providers, logger)

	routesFn := func() eu.HandlerRoutesFunc {
		return service.GetRoutes(svc)
	}

	return svc, routesFn
}
