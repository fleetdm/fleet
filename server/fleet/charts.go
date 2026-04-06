package fleet

import (
	"context"
	"time"
)

// ChartService is the bounded-context service for chart data operations.
type ChartService interface {
	// RecordUptime records a host check-in for uptime chart data.
	RecordUptime(ctx context.Context, hostID uint, timestamp time.Time) error

	// GetChartData returns time-series chart data for the given metric.
	GetChartData(ctx context.Context, metric string, opts ChartRequestOpts) (*ChartResponse, error)

	// RegisterDataset registers a chart dataset.
	RegisterDataset(ds ChartDataset)
}

// ChartDataset defines the interface for a chartable dataset.
type ChartDataset interface {
	// Name returns the dataset identifier used in the DB and API path.
	Name() string

	// Collect is called by the hourly cron job to populate bitmaps in bulk.
	// Datasets populated entirely via RecordBit on check-in can no-op.
	Collect(ctx context.Context, ds Datastore, hour time.Time) error

	// ResolveFilters translates dataset-specific query params into entity IDs.
	// Returns nil if no entity filtering is needed.
	ResolveFilters(ctx context.Context, ds Datastore, params map[string]string) ([]uint, error)

	// SupportedFilters returns metadata about available filters for this dataset.
	SupportedFilters() []ChartFilterDef

	// DefaultVisualization returns the default visualization type (e.g. "line", "heatmap").
	DefaultVisualization() string

	// HasEntityDimension returns true if the dataset uses entity_id (requiring COUNT(DISTINCT host_id)).
	HasEntityDimension() bool
}

// ChartFilterDef describes a filter that a dataset supports.
type ChartFilterDef struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "multi_select", "range", "boolean"
	Description string `json:"description,omitempty"`
}

// ChartDataPoint represents a single data point in the chart response.
type ChartDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int       `json:"value"`
}

// ChartResponse is the API response for chart data.
type ChartResponse struct {
	Metric        string           `json:"metric"`
	Visualization string           `json:"visualization"`
	TotalHosts    int              `json:"total_hosts"`
	Resolution    string           `json:"resolution"`
	Days          int              `json:"days"`
	Filters       ChartFilters     `json:"filters"`
	Data          []ChartDataPoint `json:"data"`
}

// ChartRequestOpts captures the parsed query parameters for a chart request.
type ChartRequestOpts struct {
	Days           int
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
	// DatasetFilters are dataset-specific filter params (e.g. policy_id, severity).
	DatasetFilters map[string]string
}

// ChartHostFilter is used to filter hosts in chart queries.
type ChartHostFilter struct {
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
}

// ChartFilters captures the applied filters for a chart request.
type ChartFilters struct {
	LabelIDs       []uint   `json:"label_ids,omitempty"`
	Platforms      []string `json:"platforms,omitempty"`
	IncludeHostIDs []uint   `json:"include_host_ids,omitempty"`
	ExcludeHostIDs []uint   `json:"exclude_host_ids,omitempty"`
}
