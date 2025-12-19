package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that attaches activity routes to the router.
func GetRoutes(svc activity.Service, authMiddleware endpoint.Middleware) endpoint_utils.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc activity.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption) {
	// //////////////////////////////////////////
	// User-authenticated endpoints
	ue := newUserAuthenticatedEndpointer(svc, authMiddleware, opts, r, apiVersions()...)

	// Ping endpoint: hello world for the activity bounded context
	ue.GET("/api/_version_/fleet/activity/ping", pingEndpoint, nil)

	// List activities endpoint
	ue.GET("/api/_version_/fleet/activities", listActivitiesEndpoint, listActivitiesRequest{})
}

func apiVersions() []string {
	return []string{"v1"}
}

// //////////////////////////////////////////
// Request types

type listActivitiesRequest struct {
	ListOptions    activity.ListOptions `url:"list_options"`
	ActivityType   string               `query:"activity_type,optional"`
	StartCreatedAt string               `query:"start_created_at,optional"`
	EndCreatedAt   string               `query:"end_created_at,optional"`
}

// //////////////////////////////////////////
// Endpoint handlers

func pingEndpoint(ctx context.Context, _ any, svc activity.Service) platform_http.Errorer {
	if err := svc.Ping(ctx); err != nil {
		return activity.DefaultResponse{Err: err}
	}
	return activity.PingResponse{Message: "ping"}
}

func listActivitiesEndpoint(ctx context.Context, request any, svc activity.Service) platform_http.Errorer {
	req := request.(*listActivitiesRequest)
	activities, metadata, err := svc.ListActivities(ctx, activity.ListActivitiesOptions{
		ListOptions:    req.ListOptions,
		ActivityType:   req.ActivityType,
		StartCreatedAt: req.StartCreatedAt,
		EndCreatedAt:   req.EndCreatedAt,
	})
	if err != nil {
		return activity.ListActivitiesResponse{DefaultResponse: activity.DefaultResponse{Err: err}}
	}
	return activity.ListActivitiesResponse{Meta: metadata, Activities: activities}
}
