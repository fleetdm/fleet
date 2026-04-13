// Package types provides internal types and interfaces for the chart bounded context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
)

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// GetBlobData fetches raw host bitmap blobs from host_hourly_data_blobs for a given
	// dataset and date range.
	GetBlobData(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error)

	// GetHostIDsForFilter returns the host IDs that match the given host filter.
	// Used to build a filter bitmask for blob-based datasets.
	GetHostIDsForFilter(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error)

	// CountHostsForChartFilter returns the total number of hosts matching the chart host filters.
	CountHostsForChartFilter(ctx context.Context, hostFilter *chart.HostFilter) (int, error)

	// CollectUptimeChartData bulk-inserts uptime blob data for all recently seen hosts.
	CollectUptimeChartData(ctx context.Context, now time.Time) error

	// CleanupBlobData deletes blob rows older than the specified number of days.
	CleanupBlobData(ctx context.Context, days int) error
}
