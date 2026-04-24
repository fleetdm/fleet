// Package types provides internal types and interfaces for the chart bounded context.
package types

import (
	"context"
	"time"

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

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// FindRecentlySeenHostIDs returns host IDs that have reported since the
	// given cutoff. Used by datasets like uptime that derive their sample from
	// recent host activity.
	FindRecentlySeenHostIDs(ctx context.Context, since time.Time) ([]uint, error)

	// AffectedHostIDsByCVE returns, for every CVE currently affecting any host,
	// the slice of host IDs impacted by it. Unresolved-only is implicit in the
	// underlying joins: a host's software/OS row transitions when it upgrades
	// past the vulnerable version, so the join naturally stops matching.
	AffectedHostIDsByCVE(ctx context.Context) (map[string][]uint, error)

	// TrackedCriticalCVEs returns CVE IDs matching the iteration-1 curated
	// filter: critical (CVSS >= 9.0) CVEs on a hard-coded set of software
	// titles, unioned with all critical OS vulnerabilities. Returns a non-nil
	// empty slice when nothing matches — callers pass this to GetSCDData's
	// entityIDs parameter where nil vs empty have distinct semantics.
	//
	// TODO(iteration-2): replace with user-configurable filtering.
	TrackedCriticalCVEs(ctx context.Context) ([]string, error)

	// RecordBucketData writes one or more entity bitmaps for the given bucket using
	// the specified sample strategy. See api.SampleStrategy for the semantics of
	// each strategy.
	RecordBucketData(
		ctx context.Context,
		dataset string,
		bucketStart time.Time,
		bucketSize time.Duration,
		strategy api.SampleStrategy,
		entityBitmaps map[string][]byte,
	) error

	// GetSCDData returns per-bucket distinct-host counts for a dataset over the
	// given range at the given bucket size. Aggregation within a bucket depends
	// on the sample strategy:
	//   - Accumulate: OR every row that overlaps the bucket ("hosts observed at
	//     any point during the bucket").
	//   - Snapshot: for each entity, pick the row active at bucketEnd, then OR
	//     across entities ("state as of the end of the bucket").
	// filterMask is always applied via bitmap AND — callers build it via
	// GetHostIDsForFilter + chart.HostIDsToBlob, usually through a cache.
	// The entity filter is applied via entity_id IN.
	GetSCDData(
		ctx context.Context,
		dataset string,
		startDate, endDate time.Time,
		bucketSize time.Duration,
		strategy api.SampleStrategy,
		filterMask []byte,
		entityIDs []string,
	) ([]api.DataPoint, error)

	// GetHostIDsForFilter returns the host IDs that match the given host filter.
	GetHostIDsForFilter(ctx context.Context, hostFilter *HostFilter) ([]uint, error)

	// CleanupSCDData deletes closed SCD rows whose valid_to is older than the
	// retention cutoff. Open rows (valid_to = sentinel) are never deleted.
	CleanupSCDData(ctx context.Context, days int) error
}
