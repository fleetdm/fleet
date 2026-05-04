// Package api provides the public API for the chart bounded context.
// External code should use this package to interact with the chart service.
package api

import (
	"context"
	"time"
)

// Service is the composite interface for the chart service module.
// Bootstrap returns this type.
type Service interface {
	// GetChartData returns time-series chart data for the given metric.
	GetChartData(ctx context.Context, metric string, opts RequestOpts) (*Response, error)

	// RegisterDataset registers a chart dataset.
	RegisterDataset(ds Dataset)

	// CollectDatasets runs Collect on all registered datasets for the given timestamp.
	CollectDatasets(ctx context.Context, now time.Time) error

	// CleanupData deletes chart data rows older than the specified number of days.
	CleanupData(ctx context.Context, days int) error
}

// ViewerProvider exposes authorization-relevant information about the current
// authenticated viewer. Implementations typically read the viewer context, so
// this is the seam that keeps the chart bounded context free of direct
// server/fleet imports.
type ViewerProvider interface {
	// ViewerScope reports what the authenticated viewer is allowed to see
	// across teams.
	//
	//   - isGlobal == true: the viewer has a global role and can see every
	//     team plus no-team hosts. teamIDs is ignored in that case.
	//   - isGlobal == false: the viewer is team-scoped; teamIDs lists the IDs
	//     of every team the viewer has any role on. An empty slice means the
	//     viewer authenticated but has no team memberships (unusual — they
	//     should see no hosts).
	//
	// Returns an error if no viewer is in context (should never happen behind
	// the authenticated middleware, but bounded contexts fail closed rather
	// than leak data on misconfiguration).
	ViewerScope(ctx context.Context) (isGlobal bool, teamIDs []uint, err error)
}
