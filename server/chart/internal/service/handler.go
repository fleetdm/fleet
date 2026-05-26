package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	api_http "github.com/fleetdm/fleet/v4/server/chart/api/http"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that registers chart routes on the router using the provided
// authMiddleware.
func GetRoutes(svc api.Service, authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption) {
	apiVersions := []string{"v1", "2022-04"}
	ue := newChartEndpointer(svc, authMiddleware, opts, r, apiVersions...)
	ue.GET("/api/_version_/fleet/charts/{metric}", getChartDataEndpoint, api_http.GetChartDataRequest{})
}

func getChartDataEndpoint(ctx context.Context, request any, svc api.Service) (platform_http.Errorer, error) {
	req := request.(*api_http.GetChartDataRequest)

	days := req.Days
	if days == 0 {
		days = 7
	}

	opts := api.RequestOpts{
		Days:            days,
		Resolution:      req.Resolution,
		TZOffsetMinutes: req.TZOffset,
		TeamID:          req.TeamID,
		LabelIDs:        str.ParseUintList(req.LabelIDs),
		Platforms:       str.ParseStringList(req.Platforms),
		IncludeHostIDs:  str.ParseUintList(req.IncludeHostIDs),
		ExcludeHostIDs:  str.ParseUintList(req.ExcludeHostIDs),
	}

	resp, err := svc.GetChartData(ctx, req.Metric, opts)
	if err != nil {
		return api_http.GetChartDataResponse{Err: err}, nil
	}
	return api_http.GetChartDataResponse{Response: resp}, nil
}
