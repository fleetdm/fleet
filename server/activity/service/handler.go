package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that attaches activity routes to the router.
func GetRoutes(fleetSvc fleet.Service, svc activity.Service) endpoint_utils.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, fleetSvc, svc, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, fleetSvc fleet.Service, svc activity.Service, opts []kithttp.ServerOption) {
	// //////////////////////////////////////////
	// User-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(fleetSvc, svc, opts, r, apiVersions()...)

	// Ping endpoint: hello world for the activity bounded context
	ue.GET("/api/_version_/fleet/activity/ping", pingEndpoint, nil)
}

func apiVersions() []string {
	return []string{"v1"}
}

// //////////////////////////////////////////
// Endpoint handlers

func pingEndpoint(ctx context.Context, _ any, svc activity.Service) fleet.Errorer {
	if err := svc.Ping(ctx); err != nil {
		return activity.DefaultResponse{Err: err}
	}
	return activity.PingResponse{Message: "ping"}
}
