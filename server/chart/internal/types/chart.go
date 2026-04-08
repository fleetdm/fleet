// Package types provides internal types and interfaces for the chart bounded context.
package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
)

// Datastore is the internal datastore interface for the chart bounded context.
type Datastore interface {
	// RecordHostHourlyData sets a bit in the host_hourly_data bitmap for the given host, dataset,
	// and entity. The timestamp is converted to UTC to derive the date and hour.
	RecordHostHourlyData(ctx context.Context, hostID uint, dataset string, entityID uint, timestamp time.Time) error

	// GetChartData queries the host_hourly_data table for a given dataset and date range,
	// filtered by host IDs and optional entity IDs, aggregating bitmap data into time-bucketed counts.
	GetChartData(ctx context.Context, dataset string, startDate time.Time, endDate time.Time, hostFilter *chart.HostFilter, entityIDs []uint, hasEntityDimension bool, downsample int) ([]chart.DataPoint, error)

	// CountHostsForChartFilter returns the total number of hosts matching the chart host filters.
	CountHostsForChartFilter(ctx context.Context, hostFilter *chart.HostFilter) (int, error)

	// CollectUptimeChartData bulk-inserts uptime bitmap data for all recently seen hosts.
	CollectUptimeChartData(ctx context.Context, now time.Time) error

	// CleanupHostHourlyData deletes rows older than the specified number of days.
	CleanupHostHourlyData(ctx context.Context, days int) error
}
