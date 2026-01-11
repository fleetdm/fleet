package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	api_http "github.com/fleetdm/fleet/v4/server/activity/api/http"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that registers activity routes on the router.
func GetRoutes(svc api.Service, authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption) {
	// User-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, authMiddleware, opts, r, apiVersions()...)

	ue.GET("/api/_version_/fleet/activities", listActivitiesEndpoint, api_http.ListActivitiesRequest{})
}

func apiVersions() []string {
	return []string{"v1", "latest"}
}

// listActivitiesEndpoint handles GET /api/_version_/fleet/activities
func listActivitiesEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.ListActivitiesRequest)

	// Build list options with activity-specific filters
	opt := req.ListOptions
	opt.MatchQuery = req.Query
	opt.ActivityType = req.ActivityType
	opt.StartCreatedAt = req.StartCreatedAt
	opt.EndCreatedAt = req.EndCreatedAt

	// Fill in defaults
	fillListOptions(&opt)

	activities, meta, err := svc.ListActivities(ctx, opt)
	if err != nil {
		return api_http.ListActivitiesResponse{Err: err}
	}

	return api_http.ListActivitiesResponse{
		Meta:       meta,
		Activities: activities,
	}
}
