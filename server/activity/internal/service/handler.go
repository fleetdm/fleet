package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that registers activity routes on the router.
func GetRoutes(svc api.Service, authMiddleware AuthMiddleware) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, authMiddleware AuthMiddleware, opts []kithttp.ServerOption) {
	// User-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, authMiddleware, opts, r, apiVersions()...)

	ue.GET("/api/_version_/fleet/activities", listActivitiesEndpoint, listActivitiesRequest{})
}

func apiVersions() []string {
	return []string{"v1", "latest"}
}

// Request and response types

type listActivitiesRequest struct {
	ListOptions    api.ListOptions `url:"list_options"`
	Query          string          `query:"query,optional"`
	ActivityType   string          `query:"activity_type,optional"`
	StartCreatedAt string          `query:"start_created_at,optional"`
	EndCreatedAt   string          `query:"end_created_at,optional"`
}

type listActivitiesResponse struct {
	Meta       *api.PaginationMetadata `json:"meta"`
	Activities []*api.Activity         `json:"activities"`
	Err        error                   `json:"error,omitempty"`
}

func (r listActivitiesResponse) Error() error { return r.Err }

// listActivitiesEndpoint handles GET /api/_version_/fleet/activities
func listActivitiesEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*listActivitiesRequest)

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
		return listActivitiesResponse{Err: err}
	}

	return listActivitiesResponse{
		Meta:       meta,
		Activities: activities,
	}
}
