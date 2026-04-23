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
)

// DataCollectionStateProvider resolves per-dataset on/off state from the main
// Fleet datastore. It's split out from the MySQL chart store so the chart
// bounded context stays free of direct fleet.* imports.
type DataCollectionStateProvider interface {
	DataCollectionState(ctx context.Context, dataset string) (bool, []uint, error)
}

// compositeStore satisfies types.Datastore by embedding the chart MySQL store
// and delegating DataCollectionState to an external provider.
type compositeStore struct {
	*mysql.Datastore
	dc DataCollectionStateProvider
}

func (c *compositeStore) DataCollectionState(ctx context.Context, dataset string) (bool, []uint, error) {
	return c.dc.DataCollectionState(ctx, dataset)
}

// New creates a new chart service module and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	authorizer platform_authz.Authorizer,
	viewerProvider api.ViewerProvider,
	dcProvider DataCollectionStateProvider,
	logger *slog.Logger,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := &compositeStore{Datastore: mysql.NewDatastore(dbConns, logger), dc: dcProvider}
	svc := service.NewService(authorizer, ds, viewerProvider, logger)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}
