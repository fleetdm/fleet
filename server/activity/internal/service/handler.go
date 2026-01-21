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
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/activities", listHostPastActivitiesEndpoint, api_http.ListHostPastActivitiesRequest{})
}

func apiVersions() []string {
	return []string{"v1", "latest"}
}

// listActivitiesEndpoint handles GET /api/_version_/fleet/activities
func listActivitiesEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.ListActivitiesRequest)

	opt := req.ListOptions // Access the embedded api.ListOptions
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

// listHostPastActivitiesEndpoint handles GET /api/_version_/fleet/hosts/{id}/activities
func listHostPastActivitiesEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.ListHostPastActivitiesRequest)

	opt := req.ListOptions
	fillHostPastActivitiesListOptions(&opt)

	activities, meta, err := svc.ListHostPastActivities(ctx, req.HostID, opt)
	if err != nil {
		return api_http.ListHostPastActivitiesResponse{Err: err}
	}

	return api_http.ListHostPastActivitiesResponse{
		Meta:       meta,
		Activities: activities,
	}
}

// fillHostPastActivitiesListOptions sets default values for host past activities list options.
func fillHostPastActivitiesListOptions(opt *api.ListOptions) {
	if opt.PerPage == 0 {
		opt.PerPage = defaultPerPage
	}
}

// fillListOptions sets default values for list options.
// Note: IncludeMetadata is set internally by the service layer.
func fillListOptions(opt *api.ListOptions) {
	// Default ordering by created_at descending (newest first) if not specified
	if opt.OrderKey == "" {
		opt.OrderKey = "created_at"
		opt.OrderDirection = api.OrderDescending
	}
	// Default PerPage based on whether pagination was requested
	if opt.PerPage == 0 {
		if opt.Page == 0 {
			// No pagination requested - return all results (legacy behavior)
			opt.PerPage = unlimitedPerPage
		} else {
			// Page specified without per_page - use sensible default
			opt.PerPage = defaultPerPage
		}
	}
}
