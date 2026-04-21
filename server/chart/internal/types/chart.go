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
// TeamID semantics:
//   - nil: no team filter applied (all hosts).
//   - *TeamID == 0: hosts with no team assignment (team_id IS NULL).
//   - *TeamID > 0: hosts in that team.
type HostFilter struct {
	TeamID         *uint
	LabelIDs       []uint
	Platforms      []string
	IncludeHostIDs []uint
	ExcludeHostIDs []uint
}

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// FindRecentlySeenHostIDs returns host IDs that have reported within the given
	// lookback window. Used by datasets like uptime that derive their sample from
	// recent host activity.
	FindRecentlySeenHostIDs(ctx context.Context, lookback time.Duration) ([]uint, error)

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
