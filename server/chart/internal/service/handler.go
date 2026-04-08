package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	api_http "github.com/fleetdm/fleet/v4/server/chart/api/http"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
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

func getChartDataEndpoint(ctx context.Context, request any, svc api.Service) (fleet.Errorer, error) {
	req := request.(*api_http.GetChartDataRequest)

	days := req.Days
	if days == 0 {
		days = 7
	}

	opts := chart.RequestOpts{
		Days:           days,
		Downsample:     req.Downsample,
		LabelIDs:       parseUintList(req.LabelIDs),
		Platforms:      parseStringList(req.Platforms),
		IncludeHostIDs: parseUintList(req.IncludeHostIDs),
		ExcludeHostIDs: parseUintList(req.ExcludeHostIDs),
		DatasetFilters: map[string]string{},
	}

	resp, err := svc.GetChartData(ctx, req.Metric, opts)
	if err != nil {
		return api_http.GetChartDataResponse{Err: err}, nil
	}
	return api_http.GetChartDataResponse{Response: resp}, nil
}

func parseUintList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.ParseUint(p, 10, 64); err == nil {
			result = append(result, uint(v))
		}
	}
	return result
}

func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
