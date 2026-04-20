// Package api provides the public API for the chart bounded context.
// External code should use this package to interact with the chart service.
package api

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
)

// Service is the composite interface for the chart service module.
// Bootstrap returns this type.
type Service interface {
	// GetChartData returns time-series chart data for the given metric.
	GetChartData(ctx context.Context, metric string, opts chart.RequestOpts) (*chart.Response, error)

	// RegisterDataset registers a chart dataset.
	RegisterDataset(ds chart.Dataset)

	// CollectDatasets runs Collect on all registered datasets for the given timestamp.
	CollectDatasets(ctx context.Context, now time.Time) error

	// CleanupData deletes chart data rows older than the specified number of days.
	CleanupData(ctx context.Context, days int) error
}
