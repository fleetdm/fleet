// Package chart provides the public types and interfaces for the chart bounded context.
// External code should import this package for types; implementation details are in internal/.
package chart

import (
	"context"
	"time"
)

// Dataset defines the interface for a chartable dataset.
type Dataset interface {
	// Name returns the dataset identifier used in the DB and API path.
	Name() string

	// Collect is called by the cron job to populate bitmaps in bulk.
	// Datasets populated entirely via RecordBit on check-in can no-op.
	Collect(ctx context.Context, store DatasetStore, hour time.Time) error

	// ResolveFilters translates dataset-specific query params into entity IDs.
	// Returns nil if no entity filtering is needed.
	ResolveFilters(ctx context.Context, store DatasetStore, params map[string]string) ([]uint, error)

	// SupportedFilters returns metadata about available filters for this dataset.
	SupportedFilters() []FilterDef

	// DefaultVisualization returns the default visualization type (e.g. "line", "heatmap").
	DefaultVisualization() string

	// HasEntityDimension returns true if the dataset uses entity_id (requiring COUNT(DISTINCT host_id)).
	HasEntityDimension() bool
}

// DatasetStore is the narrow interface that datasets need for their Collect and ResolveFilters methods.
// It is satisfied by the chart internal Datastore, keeping dataset implementations decoupled from internals.
type DatasetStore interface {
	CollectUptimeChartData(ctx context.Context, now time.Time) error
}

// Host is a minimal host type for authorization checks within the chart bounded context.
type Host struct{}

// AuthzType implements platform_authz.AuthzTyper.
func (h *Host) AuthzType() string { return "host" }

// FilterDef describes a filter that a dataset supports.
type FilterDef struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "multi_select", "range", "boolean"
	Description string `json:"description,omitempty"`
}

// DataPoint represents a single data point in the chart response.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int       `json:"value"`
}

// Response is the API response for chart data.
type Response struct {
	Metric        string    `json:"metric"`
	Visualization string    `json:"visualization"`
	TotalHosts    int       `json:"total_hosts"`
	Resolution    string    `json:"resolution"`
	Days          int       `json:"days"`
	Filters       Filters   `json:"filters"`
	Data          []DataPoint `json:"data"`
}

// RequestOpts captures the parsed query parameters for a chart request.
type RequestOpts struct {
	Days int
	// Downsample groups hours into N-hour blocks (valid: 0, 2, 4, 8). 0 means hourly.
	Downsample     int
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
	// DatasetFilters are dataset-specific filter params (e.g. policy_id, severity).
	DatasetFilters map[string]string
}

// HostFilter is used to filter hosts in chart queries.
type HostFilter struct {
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
}

// Filters captures the applied filters for a chart request.
type Filters struct {
	LabelIDs       []uint   `json:"label_ids,omitempty"`
	Platforms      []string `json:"platforms,omitempty"`
	IncludeHostIDs []uint   `json:"include_host_ids,omitempty"`
	ExcludeHostIDs []uint   `json:"exclude_host_ids,omitempty"`
}
