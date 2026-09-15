// Package types provides internal types and interfaces for the chart bounded context.
package types

import (
	"context"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart/api"
)

// HostFilter is the internal filter used by the service and datastore to narrow
// SCD queries to a specific set of hosts.
//
// TeamIDs semantics — the distinction between nil and empty matters:
//   - nil: no team filter applied (all hosts across all teams, including no-team).
//     This is the global-user-no-explicit-team-id case.
//   - empty non-nil ([]uint{}): caller is team-scoped but has zero accessible
//     teams. SQL falls through to a no-match clause so the user sees nothing.
//   - single 0 ([]uint{0}): hosts with no team assignment (team_id IS NULL).
//   - other values: team_id IN (list). Mixed with a 0 entry yields
//     "(team_id IS NULL OR team_id IN (non-zero list))".
type HostFilter struct {
	TeamIDs        []uint
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
}

// CVEChartFilter is the chart's CVE entity filter. It is an alias rather than
// a distinct type so the collector (which sees only the public api package) and
// the datastore agree on one filter shape, and so a filter can be keyed to its
// aggregate entity from either side.
type CVEChartFilter = api.CVEFilter

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// FindOnlineHostIDs returns host IDs that are "online right now" using a
	// platform-specific predicate. Non-mobile (osquery) hosts use the product's
	// standard online predicate: host_seen_times.seen_time within the host's own
	// check-in interval (LEAST of distributed_interval and config_tls_refresh,
	// plus a 60-second grace period that mirrors fleet.OnlineIntervalBuffer).
	// Mobile hosts (iOS, iPadOS, Android), which only check in via MDM, use
	// their MDM activity signal (nano_seen_times.seen_time, falling back to
	// detail_updated_at) within a fixed mobile online window. Used by datasets
	// like uptime.
	FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error)

	// AffectedHostIDsByCVE returns a bitmap of affected host IDs per CVE,
	// scoped to the given cves set. nil or empty cves returns an empty map.
	// Unresolved-only is implicit in the underlying joins: a host's software/OS
	// row transitions when it upgrades past the vulnerable version, so the join
	// naturally stops matching.
	AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string]*roaring.Bitmap, error)

	// CollectibleCVEs returns every CVE ID, at all severities, on the curated
	// set of tracked software (trackedCVESoftwareMatchers) unioned with all
	// operating-system vulnerabilities. This is the wide set the CVE collector
	// records into host_scd_data; display-time narrowing happens at read time
	// via ResolveCVEChartEntities. Returns a non-nil empty slice when nothing
	// matches.
	CollectibleCVEs(ctx context.Context) ([]string, error)

	// ResolveCVEChartEntities resolves the read-time CVE allow-set for the chart
	// by intersecting the curated universe with the filter's predicates
	// (category, CVSS range, EPSS range, known-exploit) and subtracting any
	// excluded CVEs. Returns a non-nil empty slice when the filter resolves to
	// nothing — callers pass this to GetSCDData's entityIDs parameter, never
	// nil, so lower-severity CVEs never leak into the chart.
	ResolveCVEChartEntities(ctx context.Context, filter CVEChartFilter) ([]string, error)

	// AggregateCoversFrom reports whether an aggregate entity's series can
	// answer a request starting at `from`. A series is built backwards over
	// several ticks, so the read path must check how far it reaches rather than
	// merely that it exists. horizon is the oldest hour retention still
	// guarantees; a series reaching it holds everything any path could answer.
	AggregateCoversFrom(ctx context.Context, dataset, entityID string, from, horizon time.Time) (bool, error)

	// BackfillAggregateEntity extends an aggregate entity's history one bounded
	// batch further back, reconstructing it from the rows already collected for
	// sourceIDs, never reaching below horizon. Reports whether it wrote a batch;
	// false means there is nothing left to reconstruct.
	BackfillAggregateEntity(ctx context.Context, dataset, aggregateID string, sourceIDs []string, now, horizon time.Time) (bool, error)

	// RecordBucketData writes one or more entity bitmaps for the given bucket using
	// the specified sample strategy. See api.SampleStrategy for the semantics of
	// each strategy. Bitmaps are passed in op form (*roaring.Bitmap); the
	// datastore serializes via chart.BitmapToBlob at the storage boundary.
	RecordBucketData(
		ctx context.Context,
		dataset string,
		bucketStart time.Time,
		bucketSize time.Duration,
		strategy api.SampleStrategy,
		entityBitmaps map[string]*roaring.Bitmap,
	) error

	// GetSCDData returns per-bucket distinct-host counts for a dataset over the
	// given range at the given bucket size. Aggregation within a bucket depends
	// on the sample strategy:
	//   - Accumulate: OR every row that overlaps the bucket ("hosts observed at
	//     any point during the bucket").
	//   - Snapshot: for each entity, pick the row active at bucketEnd, then OR
	//     across entities ("state as of the end of the bucket").
	// filterMask is always applied via bitmap AND — callers build it via
	// GetHostIDsForFilter + chart.NewBitmap, usually through a cache.
	// The entity filter is applied via entity_id IN.
	GetSCDData(
		ctx context.Context,
		dataset string,
		startDate, endDate time.Time,
		bucketSize time.Duration,
		strategy api.SampleStrategy,
		filterMask *roaring.Bitmap,
		entityIDs []string,
	) ([]api.DataPoint, error)

	// GetHostIDsForFilter returns the host IDs that match the given host filter.
	GetHostIDsForFilter(ctx context.Context, hostFilter *HostFilter) ([]uint, error)

	// CleanupSCDData deletes closed SCD rows whose valid_to is older than the
	// retention cutoff. Open rows (valid_to = sentinel) are never deleted.
	CleanupSCDData(ctx context.Context, days int) error

	// DeleteAllForDataset removes every host_scd_data row whose dataset column
	// matches `dataset`, in batches of up to `batchSize` rows per statement,
	// looping until no rows remain. Used by the global scrub worker when an
	// admin disables a dataset entirely. Each batch is its own transaction so
	// long-running deletes don't hold locks for unbounded durations.
	DeleteAllForDataset(ctx context.Context, dataset string, batchSize int) error

	// HostIDsInFleets returns host IDs whose team_id is in fleetIDs. Used by
	// the per-fleet scrub worker to build the bit mask of hosts to clear from
	// existing host_scd_data rows. Returns nil/empty for empty input.
	HostIDsInFleets(ctx context.Context, fleetIDs []uint) ([]uint, error)

	// ApplyScrubMaskToDataset walks every host_scd_data row for the given
	// dataset in id-order with `batchSize`-row pages, computing
	// chart.BlobANDNOT(host_bitmap, mask) and writing the result back via
	// UPDATE. Used by the per-fleet scrub worker.
	ApplyScrubMaskToDataset(ctx context.Context, dataset string, mask *roaring.Bitmap, batchSize int) error
}
