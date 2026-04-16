// Package http provides HTTP request/response types for the chart bounded context.
package http

import "github.com/fleetdm/fleet/v4/server/chart"

// GetChartDataRequest is the HTTP request for the chart data endpoint.
type GetChartDataRequest struct {
	Metric         string `url:"metric"`
	Days           int    `query:"days,optional"`
	Downsample     int    `query:"downsample,optional"`
	TZOffset       int    `query:"tz_offset,optional"`
	LabelIDs       string `query:"label_ids,optional"`
	Platforms      string `query:"platforms,optional"`
	IncludeHostIDs string `query:"include_host_ids,optional"`
	ExcludeHostIDs string `query:"exclude_host_ids,optional"`
}

// GetChartDataResponse is the HTTP response for the chart data endpoint.
type GetChartDataResponse struct {
	*chart.Response
	Err error `json:"error,omitempty"`
}

func (r GetChartDataResponse) Error() error { return r.Err }

// GetMostIgnoredPoliciesRequest is the HTTP request for the most-ignored
// policies endpoint.
type GetMostIgnoredPoliciesRequest struct {
	Limit int `query:"limit,optional"`
}

// GetMostIgnoredPoliciesResponse is the HTTP response for the most-ignored
// policies endpoint.
type GetMostIgnoredPoliciesResponse struct {
	Policies []chart.MostIgnoredPolicy `json:"policies"`
	Err      error                     `json:"error,omitempty"`
}

func (r GetMostIgnoredPoliciesResponse) Error() error { return r.Err }

// GetTopNonCompliantHostsRequest is the HTTP request for the ranked host list.
type GetTopNonCompliantHostsRequest struct {
	Limit int `query:"limit,optional"`
}

// GetTopNonCompliantHostsResponse is the HTTP response for the ranked host list.
type GetTopNonCompliantHostsResponse struct {
	Hosts []chart.HostFailingSummary `json:"hosts"`
	Err   error                      `json:"error,omitempty"`
}

func (r GetTopNonCompliantHostsResponse) Error() error { return r.Err }
