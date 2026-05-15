// Package bootstrap provides the public entry point for the chart bounded context.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/chart/internal/service"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-kit/kit/endpoint"
	"github.com/jmoiron/sqlx"
)

// New creates a new chart service module and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	authorizer platform_authz.Authorizer,
	viewerProvider api.ViewerProvider,
	logger *slog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(authorizer, ds, viewerProvider, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}

// TrackedCriticalCVEs returns the curated set of CVE IDs that the chart
// collector currently tracks. Exposed for development tools (e.g.
// charts-backfill) that need to mirror the production CVE-selection logic
// without constructing the full bounded context.
func TrackedCriticalCVEs(ctx context.Context, db *sqlx.DB, logger *slog.Logger) ([]string, error) {
	ds := mysql.NewDatastore(&platform_mysql.DBConnections{Primary: db, Replica: db}, logger)
	return ds.TrackedCriticalCVEs(ctx)
}
