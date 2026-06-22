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

	// CollectDatasets runs Collect on registered datasets for the given timestamp.
	//
	// `scope` is a function that takes a dataset name and returns two values:
	// 1. `skip` (bool): Indicates whether the dataset should be skipped entirely.
	//     If `true`, the service will not call the Collect method for this dataset.
	// 2. `disabledFleetIDs` ([]uint): A slice of fleet IDs that are disabled for this dataset.
	//     This is forwarded to the Collect method for each dataset.
	//
	// If `scope` is nil, it is treated as a function that returns (false, nil) for every dataset.
	CollectDatasets(ctx context.Context, now time.Time, scope CollectScopeFn) error

	// CleanupData deletes chart data rows older than the specified number of days.
	CleanupData(ctx context.Context, days int) error

	// ScrubDatasetGlobal removes every collected row for the given dataset.
	// Invoked by the chart_scrub_dataset_global worker after an admin disables
	// the dataset globally. Idempotent — a retry on partially-completed work
	// converges to the same end state (no rows for the dataset).
	ScrubDatasetGlobal(ctx context.Context, dataset string) error

	// ScrubDatasetFleet clears bits for every host currently in any of
	// fleetIDs from every host_scd_data row for the dataset. Invoked by the
	// chart_scrub_dataset_fleet worker after an admin disables the dataset
	// for one or more fleets in a single operation. Idempotent.
	ScrubDatasetFleet(ctx context.Context, dataset string, fleetIDs []uint) error
}

// CollectScopeFn resolves per-dataset collection scope. See Service.CollectDatasets.
type CollectScopeFn func(datasetName string) (skip bool, disabledFleetIDs []uint)

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
