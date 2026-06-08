package api

import (
	"context"
	"time"

	"github.com/RoaringBitmap/roaring"
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

	// DefaultResolutionHours returns the default display granularity in hours.
	// Used when the caller doesn't specify RequestOpts.Resolution. Unrelated
	// to write-side granularity — all collectors write at 1h regardless of
	// display resolution; see SampleStrategy for details.
	DefaultResolutionHours() int

	// SampleStrategy returns how samples combine within and across buckets.
	SampleStrategy() SampleStrategy

	// Collect is called by the cron job to populate data in bulk.
	//
	// disabledFleetIDs scopes which fleets contribute to this collection. The
	// orchestrator derives it from per-team config (teams whose Enabled(name)
	// is false). Implementations should use this to filter out hosts from
	// disabled fleets when collecting data.
	//
	// No-team hosts (team_id IS NULL) are always included when the orchestrator
	// invokes Collect — the orchestrator skips Collect entirely if the global
	// flag is off.
	Collect(ctx context.Context, store DatasetStore, now time.Time, disabledFleetIDs []uint) error

	// DefaultVisualization returns the default visualization type (e.g. "line", "heatmap").
	DefaultVisualization() string
}

// DatasetStore is the narrow interface that datasets need for their Collect
// method. It is satisfied by the chart internal Datastore, keeping dataset
// implementations decoupled from internals.
type DatasetStore interface {
	// FindOnlineHostIDs returns host IDs that are "online right now" per the
	// product's standard online predicate (host_seen_times.seen_time within
	// the host's own check-in interval). MDM-only mobile devices (iOS,
	// iPadOS, Android) are excluded by design — they don't have
	// host_seen_times rows. Used by datasets like uptime.
	FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error)

	// AffectedHostIDsByCVE returns host IDs grouped by CVE, scoped to the given
	// cves set. nil or empty cves returns an empty map — callers must pass the
	// CVE set they want to collect for. Unresolved-only is implicit in the
	// underlying joins: a host's software/OS row transitions when it upgrades
	// past the vulnerable version, so the join naturally stops matching.
	AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string][]uint, error)

	// TrackedCriticalCVEs returns CVE IDs matching the iteration-1 curated
	// filter: critical (CVSS >= 9.0) CVEs on a hard-coded set of software
	// titles, unioned with all critical OS vulnerabilities. Used by the CVE
	// collector to scope collection to only the CVEs the chart actually
	// renders. See TODO in the mysql implementation.
	TrackedCriticalCVEs(ctx context.Context) ([]string, error)

	// RecordBucketData writes one or more entity bitmaps for the given bucket
	// using the specified sample strategy. See SampleStrategy for semantics.
	// Bitmaps are passed in op form (*roaring.Bitmap); the datastore
	// serializes via chart.BitmapToBlob at the storage boundary.
	RecordBucketData(
		ctx context.Context,
		dataset string,
		bucketStart time.Time,
		bucketSize time.Duration,
		strategy SampleStrategy,
		entityBitmaps map[string]*roaring.Bitmap,
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
	// Resolution is the display granularity in hours. Must be 0 or a positive
	// divisor of 24. 0 means "use the dataset's default resolution."
	Resolution int
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
}

// Filters captures the applied filters for a chart request.
type Filters struct {
	TeamID         *uint    `json:"fleet_id,omitempty"`
	LabelIDs       []uint   `json:"label_ids,omitempty"`
	Platforms      []string `json:"platforms,omitempty"`
	IncludeHostIDs []uint   `json:"include_host_ids,omitempty"`
	ExcludeHostIDs []uint   `json:"exclude_host_ids,omitempty"`
}
