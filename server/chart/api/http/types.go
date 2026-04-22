// Package http provides HTTP request/response types for the chart bounded context.
package http

import "github.com/fleetdm/fleet/v4/server/chart/api"

// GetChartDataRequest is the HTTP request for the chart data endpoint.
type GetChartDataRequest struct {
	Metric     string `url:"metric"`
	Days       int    `query:"days,optional"`
	Resolution int    `query:"resolution,optional"`
	TZOffset   int    `query:"tz_offset,optional"`
	// TeamID is a pointer so we can distinguish "absent" (auto-scope to the
	// viewer) from fleet_id=0 (no-team hosts, a valid Fleet filter).
	// Exposed as fleet_id on the wire per the teams→fleets rename; the Go
	// field name stays TeamID to match the rest of the codebase.
	TeamID         *uint  `query:"fleet_id,optional"`
	LabelIDs       string `query:"label_ids,optional"`
	Platforms      string `query:"platforms,optional"`
	IncludeHostIDs string `query:"include_host_ids,optional"`
	ExcludeHostIDs string `query:"exclude_host_ids,optional"`
}

// GetChartDataResponse is the HTTP response for the chart data endpoint.
type GetChartDataResponse struct {
	*api.Response
	Err error `json:"error,omitempty"`
}

func (r GetChartDataResponse) Error() error { return r.Err }
