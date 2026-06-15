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

	// CVE entity filters (apply only to the cve metric). Comma-separated lists
	// for categories/CVEs; EPSS and severity bounds are scalar pointers so an
	// absent bound stays nil. EPSS values are 0.0–1.0.
	SoftwareFilters string   `query:"software_filters,optional"`
	KnownExploit    bool     `query:"has_known_exploit,optional"`
	EPSSMin         *float64 `query:"epss_min,optional"`
	EPSSMax         *float64 `query:"epss_max,optional"`
	SeverityMin     *float64 `query:"severity_min,optional"`
	SeverityMax     *float64 `query:"severity_max,optional"`
	ExcludeCVEs     string   `query:"exclude_vulnerabilities,optional"`
}

// GetChartDataResponse is the HTTP response for the chart data endpoint.
type GetChartDataResponse struct {
	*api.Response
	Err error `json:"error,omitempty"`
}

func (r GetChartDataResponse) Error() error { return r.Err }
