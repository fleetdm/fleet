package api

import (
	"context"
	"time"
)

// SampleStrategy describes how a dataset's samples combine within a bucket and
// whether rows can collapse across buckets when the bitmap is unchanged.
type SampleStrategy string

const (
	// SampleStrategyAccumulate means each sample is a partial observation.
	// Writes: every row is born closed (valid_to set at insert time to bucketEnd).
	// Within-bucket samples OR-merge into the existing row via ODKU; a sample in
	// a new bucket just creates a new row with a new valid_from. No explicit
	// close step, no cross-bucket collapse.
	// Reads: bucket value = OR of every row whose interval overlaps the bucket
	// ("hosts observed at any point during the bucket").
	// Used for datasets like uptime and software usage.
	// @todo: implement job to collapse identical consecutive rows
	//        to optimize storage and query performance.
	SampleStrategyAccumulate SampleStrategy = "accumulate"

	// SampleStrategySnapshot means each sample is the full state of a single moment.
	// Writes: rows are always keyed to 1h boundaries (so row transitions align
	// to hour marks regardless of tz). Within a 1h write-bucket, the latest
	// sample's bitmap overwrites via ODKU — last sample wins. Across buckets,
	// unchanged state keeps the row open (valid_to = sentinel); a changed sample
	// closes the prior row at the new hour boundary and opens a new one.
	// Reads: bucket value = OR across entities of each entity's row active at
	// bucketEnd ("state as of the end of the bucket"). An entity whose row was
	// closed mid-bucket with no replacement is absent at bucketEnd.
	// Used for datasets like CVE and software inventory.
	SampleStrategySnapshot SampleStrategy = "snapshot"
)

// Dataset defines the interface for a chartable dataset.
type Dataset interface {
	// Name returns the dataset identifier used in the DB and API path.
	Name() string

	// BucketSize returns the default display granularity for this dataset (e.g.
	// time.Hour for uptime, 24*time.Hour for CVE). The chart walker queries at
	// this granularity by default. Write-side granularity is a separate concern
	// handled by each strategy's collector (snapshot always writes at 1h; see
	// SampleStrategy for details).
	BucketSize() time.Duration

	// SampleStrategy returns how samples combine within and across buckets.
	SampleStrategy() SampleStrategy

	// Collect is called by the cron job to populate data in bulk.
	Collect(ctx context.Context, store DatasetStore, now time.Time) error

	// ResolveFilters translates dataset-specific query params into entity IDs.
	// Returns nil if no entity filtering is needed.
	ResolveFilters(ctx context.Context, store DatasetStore, params map[string]string) ([]string, error)

	// SupportedFilters returns metadata about available filters for this dataset.
	SupportedFilters() []FilterDef

	// DefaultVisualization returns the default visualization type (e.g. "line", "heatmap").
	DefaultVisualization() string

	// HasEntityDimension returns true if the dataset partitions data by entity_id
	// (e.g. per-CVE or per-policy). Used by query paths that need to aggregate across
	// or filter by entities.
	HasEntityDimension() bool
}

// DatasetStore is the narrow interface that datasets need for their Collect and
// ResolveFilters methods. It is satisfied by the chart internal Datastore,
// keeping dataset implementations decoupled from internals.
type DatasetStore interface {
	// FindRecentlySeenHostIDs returns host IDs that have reported within the given
	// lookback window. Used by datasets like uptime that derive their sample from
	// recent host activity.
	FindRecentlySeenHostIDs(ctx context.Context, lookback time.Duration) ([]uint, error)

	// RecordBucketData writes one or more entity bitmaps for the given bucket
	// using the specified sample strategy. See SampleStrategy for semantics.
	RecordBucketData(
		ctx context.Context,
		dataset string,
		bucketStart time.Time,
		bucketSize time.Duration,
		strategy SampleStrategy,
		entityBitmaps map[string][]byte,
	) error
}

// Host is a minimal host type for authorization checks within the chart bounded context.
// The JSON tags matter: the OPA rego policy reads object.team_id via the JSON-encoded
// input, so renaming or dropping the tag silently breaks team-scoped authorization.
type Host struct {
	ID     uint  `json:"id"`
	TeamID *uint `json:"team_id"`
}

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
	Metric        string      `json:"metric"`
	Visualization string      `json:"visualization"`
	TotalHosts    int         `json:"total_hosts"`
	Resolution    string      `json:"resolution"`
	Days          int         `json:"days"`
	Filters       Filters     `json:"filters"`
	Data          []DataPoint `json:"data"`
}

// RequestOpts captures the parsed query parameters for a chart request.
type RequestOpts struct {
	Days int
	// Downsample groups hours into N-hour blocks. Must be 0 or a positive
	// divisor of 24. Both 0 (default) and 1 mean no downsampling (hourly data).
	Downsample int
	// TZOffsetMinutes is the client's UTC offset as reported by JavaScript's
	// Date.getTimezoneOffset() (positive = west of UTC, e.g. CDT = 300).
	// Used to align hourly bucket boundaries to local time.
	TZOffsetMinutes int
	// TeamID scopes the request to a single team. nil = global (authz + data
	// both fall back to the user's accessible scope). *TeamID == 0 means
	// hosts with no team assignment, matching Fleet's convention elsewhere.
	TeamID         *uint
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
	// DatasetFilters are dataset-specific filter params (e.g. policy_id, severity).
	DatasetFilters map[string]string
}

// Filters captures the applied filters for a chart request.
type Filters struct {
	TeamID         *uint    `json:"team_id,omitempty"`
	LabelIDs       []uint   `json:"label_ids,omitempty"`
	Platforms      []string `json:"platforms,omitempty"`
	IncludeHostIDs []uint   `json:"include_host_ids,omitempty"`
	ExcludeHostIDs []uint   `json:"exclude_host_ids,omitempty"`
}
