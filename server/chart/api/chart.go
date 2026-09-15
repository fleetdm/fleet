package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
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
	// FindOnlineHostIDs returns host IDs that are "online right now" using a
	// platform-specific predicate. Non-mobile (osquery) hosts use the product's
	// standard online predicate (host_seen_times.seen_time within the host's own
	// check-in interval). Mobile hosts (iOS, iPadOS, Android), which only check
	// in via MDM, use their MDM activity signal (nano_seen_times.seen_time,
	// falling back to detail_updated_at) within a fixed mobile online window.
	// Used by datasets like uptime.
	FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error)

	// AffectedHostIDsByCVE returns a bitmap of affected host IDs per CVE,
	// scoped to the given cves set. nil or empty cves returns an empty map —
	// callers must pass the CVE set they want to collect for. Unresolved-only
	// is implicit in the underlying joins: a host's software/OS row transitions
	// when it upgrades past the vulnerable version, so the join naturally
	// stops matching. Bitmaps are returned in op form, ready to pass to
	// RecordBucketData.
	AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string]*roaring.Bitmap, error)

	// CollectibleCVEs returns every CVE ID, at all severities, on the curated
	// set of tracked software unioned with all operating-system vulnerabilities.
	// Used by the CVE collector to scope collection. Display-time narrowing
	// (severity, category, EPSS, etc.) happens later at read time, so the
	// collector deliberately records the wide set. See the mysql implementation.
	CollectibleCVEs(ctx context.Context) ([]string, error)

	// ResolveCVEChartEntities resolves a filter to the CVE IDs it selects. The
	// collector uses it to precompute a filter's union; the read path uses it
	// to scope a per-CVE request. Both go through the same resolver so a
	// precomputed series and a live one always cover the same CVEs.
	ResolveCVEChartEntities(ctx context.Context, filter CVEFilter) ([]string, error)

	// BackfillAggregateEntity extends an aggregate entity's history one bounded
	// batch further back, reconstructing it from the rows already collected for
	// sourceIDs, so rebuilding a retention window is spread over several ticks
	// rather than stalling one. It never reaches below horizon, the oldest hour
	// retention still guarantees. Reports whether it wrote a batch, letting the
	// caller budget how much rebuilding a single tick does; false means there
	// is nothing left to reconstruct.
	BackfillAggregateEntity(ctx context.Context, dataset, aggregateID string, sourceIDs []string, now, horizon time.Time) (bool, error)

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

// MetricCVE is the metric name of the vulnerability-exposure (CVE) dataset.
// The CVE entity filters apply only to this metric.
const MetricCVE = "cve"

// RetentionDays is how long collected rows are kept. Cleanup deletes closed
// rows older than this, and the aggregate backfill stops at the same horizon,
// since rows below it may already be gone on every path.
const RetentionDays = 30

// CVE chart software category keys. These are the API contract for the
// `software_filters` query parameter and are mirrored by the frontend. The
// "os" category covers both operating-system vulnerabilities and the kernel
// software matchers.
const (
	CVECategoryOS       = "os"
	CVECategoryBrowsers = "browsers"
	CVECategoryOffice   = "office"
	CVECategoryAdobe    = "adobe"
)

// CVEFilter narrows the CVE chart entity set to a resolved allow-set of CVE
// IDs. All predicates AND together (intersect); ExcludeCVEs are subtracted
// afterward. Excluding a CVE that isn't in the set is a harmless no-op.
//
// Categories empty means "all categories" (no narrowing). CVSSMin/CVSSMax and
// EPSSMin/EPSSMax are nil when no bound was requested, which drops the
// corresponding predicate entirely rather than substituting the full range.
// That distinction is load-bearing for CVSS: cve_meta.cvss_score is nullable,
// so a 0.0-10.0 bound still excludes CVEs with no score, while a nil bound
// includes them. CVSS values are 0.0-10.0; EPSS values are 0.0-1.0 to match
// cve_meta.epss_probability.
type CVEFilter struct {
	Categories   []string
	CVSSMin      *float64
	CVSSMax      *float64
	EPSSMin      *float64
	EPSSMax      *float64
	KnownExploit bool
	ExcludeCVEs  []string
}

// AggregateEntityPrefix namespaces the entity IDs holding precomputed unions.
// Real CVE IDs start with "CVE-", so the two can never collide.
const AggregateEntityPrefix = "agg:"

// 128 bits of digest: far beyond collision range, and well inside entity_id's
// 100-character column.
const aggregateEntityIDHexLen = 32

var allCVECategories = []string{CVECategoryAdobe, CVECategoryBrowsers, CVECategoryOffice, CVECategoryOS}

// AggregateEntityID returns the host_scd_data entity ID holding this filter's
// precomputed per-bucket union. Keyed by contents, not by role, so filters
// resolving to the same CVE set share a series and a changed filter starts a
// new one rather than redefining the old.
func (f CVEFilter) AggregateEntityID() string {
	sum := sha256.Sum256([]byte(f.canonical()))
	return AggregateEntityPrefix + hex.EncodeToString(sum[:])[:aggregateEntityIDHexLen]
}

// canonical renders the filter as a stable string. Field labels keep equal
// values on different fields (a 5.0 CVSS floor vs a 5.0 ceiling) distinct.
func (f CVEFilter) canonical() string {
	var b strings.Builder
	b.WriteString("categories=")
	b.WriteString(canonicalCategories(f.Categories))
	b.WriteString("\ncvss_min=")
	b.WriteString(canonicalBound(f.CVSSMin))
	b.WriteString("\ncvss_max=")
	b.WriteString(canonicalBound(f.CVSSMax))
	b.WriteString("\nepss_min=")
	b.WriteString(canonicalBound(f.EPSSMin))
	b.WriteString("\nepss_max=")
	b.WriteString(canonicalBound(f.EPSSMax))
	b.WriteString("\nknown_exploit=")
	b.WriteString(strconv.FormatBool(f.KnownExploit))
	b.WriteString("\nexclude=")
	b.WriteString(strings.Join(sortedUnique(f.ExcludeCVEs), ","))
	return b.String()
}

// canonicalCategories reduces a selection to what the resolver acts on.
// Unrecognized keys select no matcher, so they are dropped. Every known
// category narrows nothing, so it collapses onto the empty selection's marker.
// A non-empty selection with no known key stays distinct from both: it resolves
// to no CVEs rather than to all of them.
func canonicalCategories(categories []string) string {
	if len(categories) == 0 {
		return "*"
	}
	known := make([]string, 0, len(categories))
	for _, c := range sortedUnique(categories) {
		if slices.Contains(allCVECategories, c) {
			known = append(known, c)
		}
	}
	if len(known) == len(allCVECategories) {
		return "*"
	}
	return strings.Join(known, ",")
}

// canonicalBound renders an optional score bound. Absent and zero must differ:
// absent drops the predicate and admits unscored CVEs, zero keeps it and
// excludes them.
func canonicalBound(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'g', -1, 64)
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
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

	// CVE entity filters (apply only to the MetricCVE metric).
	SoftwareFilters []string
	KnownExploit    bool
	// EPSS bounds are 0.0–1.0 (matching cve_meta.epss_probability); nil means
	// no bound. The frontend converts its 0–100 % input before sending.
	EPSSMin *float64
	EPSSMax *float64
	// Severity bounds are CVSS v3 base scores, 0.0–10.0; nil means no bound on
	// that side, which drops the predicate rather than widening it to the full
	// range.
	SeverityMin *float64
	SeverityMax *float64
	// ExcludeCVEs is a subtractive filter — these CVEs are removed from the
	// resolved entity set.
	ExcludeCVEs []string
}

// Filters captures the applied filters for a chart request.
type Filters struct {
	TeamID         *uint    `json:"fleet_id,omitempty"`
	LabelIDs       []uint   `json:"label_ids,omitempty"`
	Platforms      []string `json:"platforms,omitempty"`
	IncludeHostIDs []uint   `json:"include_host_ids,omitempty"`
	ExcludeHostIDs []uint   `json:"exclude_host_ids,omitempty"`

	SoftwareFilters []string `json:"software_filters,omitempty"`
	KnownExploit    bool     `json:"has_known_exploit,omitempty"`
	EPSSMin         *float64 `json:"epss_min,omitempty"`
	EPSSMax         *float64 `json:"epss_max,omitempty"`
	SeverityMin     *float64 `json:"severity_min,omitempty"`
	SeverityMax     *float64 `json:"severity_max,omitempty"`
	ExcludeCVEs     []string `json:"exclude_vulnerabilities,omitempty"`
}
