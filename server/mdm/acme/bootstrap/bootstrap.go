// Package bootstrap provides the public entry point for the ACME service module.
// It wires together internal components and exposes them for use in serve.go.
package bootstrap

import (
	"crypto/x509"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-kit/kit/endpoint"
)

type ServiceOption = service.ServiceOption

// New creates a new ACME service module and returns its service and route handler.
func New(
	dbConns *platform_mysql.DBConnections,
	redisPool acme.RedisPool,
	providers acme.DataProviders,
	logger *slog.Logger,
	opts ...ServiceOption,
) (api.Service, func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc) {
	ds := mysql.NewDatastore(dbConns, logger)
	svc := service.NewService(ds, redisPool, providers, logger, opts...)

	routesFn := func(authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
		return service.GetRoutes(svc, authMiddleware)
	}

	return svc, routesFn
}

func WithTestAppleRootCAs(rootCAs *x509.CertPool) ServiceOption {
	return func(svc *service.Service) {
		svc.TestAppleRootCAs = rootCAs
	}
}
