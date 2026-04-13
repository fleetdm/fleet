// Package service provides the service implementation for the chart bounded context.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// Service is the chart bounded context service implementation.
type Service struct {
	authz    platform_authz.Authorizer
	store    types.Datastore
	datasets map[string]chart.Dataset
	logger   *slog.Logger
}

// NewService creates a new chart service.
func NewService(authz platform_authz.Authorizer, store types.Datastore, logger *slog.Logger) *Service {
	return &Service{
		authz:    authz,
		store:    store,
		datasets: make(map[string]chart.Dataset),
		logger:   logger,
	}
}

// Ensure Service implements api.Service at compile time.
var _ api.Service = (*Service)(nil)

func (s *Service) RegisterDataset(ds chart.Dataset) {
	s.datasets[ds.Name()] = ds
}

func (s *Service) CollectDatasets(ctx context.Context, now time.Time) error {
	for name, dataset := range s.datasets {
		if err := dataset.Collect(ctx, s.store, now); err != nil {
			// Log and continue — don't let one dataset failure block others.
			logging.WithErr(ctx, ctxerr.Wrap(ctx, err, fmt.Sprintf("collect chart dataset %s", name)))
		}
	}
	return nil
}

func (s *Service) GetChartData(ctx context.Context, metric string, opts chart.RequestOpts) (*chart.Response, error) {
	if err := s.authz.Authorize(ctx, &chart.Host{}, platform_authz.ActionRead); err != nil {
		return nil, err
	}

	dataset, ok := s.datasets[metric]
	if !ok {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("unknown chart metric: %s", metric)}
	}

	// Validate days preset.
	validDays := map[int]struct{}{1: {}, 7: {}, 14: {}, 30: {}}
	if _, ok := validDays[opts.Days]; !ok {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("invalid days value: %d (must be 1, 7, 14, or 30)", opts.Days)}
	}

	// Validate downsample.
	validDownsample := map[int]struct{}{0: {}, 2: {}, 4: {}, 8: {}}
	if _, ok := validDownsample[opts.Downsample]; !ok {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("invalid downsample value: %d (must be 2, 4, or 8)", opts.Downsample)}
	}

	// Resolve dataset-specific filters to entity IDs.
	entityIDs, err := dataset.ResolveFilters(ctx, s.store, opts.DatasetFilters)
	if err != nil {
		return nil, err
	}

	// Calculate date range — go back exactly N days from now.
	now := time.Now().UTC()
	endDate := now
	startDate := now.AddDate(0, 0, -opts.Days)

	// Build host filter.
	var hostFilter *chart.HostFilter
	if len(opts.LabelIDs) > 0 || len(opts.Platforms) > 0 || len(opts.IncludeHostIDs) > 0 || len(opts.ExcludeHostIDs) > 0 {
		hostFilter = &chart.HostFilter{
			LabelIDs:       opts.LabelIDs,
			Platforms:      opts.Platforms,
			IncludeHostIDs: opts.IncludeHostIDs,
			ExcludeHostIDs: opts.ExcludeHostIDs,
		}
	}

	var data []chart.DataPoint
	var totalHosts int
	resolution := "hourly"
	if opts.Downsample > 0 {
		resolution = fmt.Sprintf("%d-hour", opts.Downsample)
	}

	switch dataset.StorageType() {
	case chart.StorageTypeBlob:
		data, totalHosts, err = s.getChartDataBlob(ctx, metric, startDate, endDate, hostFilter, entityIDs, opts.Downsample)
		if err == nil {
			data = fillZeroValues(data, startDate, endDate, opts.Downsample)
		}
	case chart.StorageTypeSCD:
		// SCD datasets always bucket daily — the CTE fills zero buckets itself.
		const bucketHours = 24
		resolution = "daily"
		data, err = s.store.GetSCDData(ctx, metric, startDate, endDate, bucketHours, hostFilter, entityIDs)
		if err == nil {
			totalHosts, err = s.store.CountHostsForChartFilter(ctx, hostFilter)
		}
	default:
		return nil, ctxerr.Errorf(ctx, "unsupported storage type: %s", dataset.StorageType())
	}
	if err != nil {
		return nil, err
	}

	return &chart.Response{
		Metric:        metric,
		Visualization: dataset.DefaultVisualization(),
		TotalHosts:    totalHosts,
		Resolution:    resolution,
		Days:          opts.Days,
		Filters: chart.Filters{
			LabelIDs:       opts.LabelIDs,
			Platforms:      opts.Platforms,
			IncludeHostIDs: opts.IncludeHostIDs,
			ExcludeHostIDs: opts.ExcludeHostIDs,
		},
		Data: data,
	}, nil
}

// getChartDataBlob fetches raw blobs from the datastore and aggregates them in Go.
// It handles host filtering (via bitwise AND) and downsampling (via bitwise OR of hour groups).
func (s *Service) getChartDataBlob(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	hostFilter *chart.HostFilter,
	entityIDs []string,
	downsample int,
) ([]chart.DataPoint, int, error) {
	blobs, err := s.store.GetBlobData(ctx, dataset, startDate, endDate, entityIDs)
	if err != nil {
		return nil, 0, err
	}

	// Build filter mask if host filters are present.
	var filterMask []byte
	var totalHosts int
	if hostFilter != nil {
		hostIDs, err := s.store.GetHostIDsForFilter(ctx, hostFilter)
		if err != nil {
			return nil, 0, err
		}
		totalHosts = len(hostIDs)
		filterMask = chart.HostIDsToBlob(hostIDs)
	} else {
		var err error
		totalHosts, err = s.store.CountHostsForChartFilter(ctx, nil)
		if err != nil {
			return nil, 0, err
		}
	}

	step := 1
	if downsample > 0 {
		step = downsample
	}

	// Index blobs by (date, hour) for efficient lookup.
	type dateHourKey struct {
		date string
		hour int
	}
	blobIndex := make(map[dateHourKey][]byte, len(blobs))
	for _, b := range blobs {
		key := dateHourKey{date: b.ChartDate.Format("2006-01-02"), hour: b.Hour}
		blobIndex[key] = b.HostBitmap
	}

	// Build data points: for each step-aligned hour bucket, OR the blobs in the window,
	// optionally AND with filter mask, then popcount.
	var results []chart.DataPoint
	for h := 0; h+step <= 24; h += step {
		// Collect all dates in the range.
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")

			// OR blobs within the downsample window.
			var merged []byte
			for offset := range step {
				key := dateHourKey{date: dateStr, hour: h + offset}
				if blob, ok := blobIndex[key]; ok {
					merged = chart.BlobOR(merged, blob)
				}
			}

			// Apply host filter.
			if filterMask != nil && merged != nil {
				merged = chart.BlobAND(merged, filterMask)
			}

			count := chart.BlobPopcount(merged)
			ts := time.Date(d.Year(), d.Month(), d.Day(), h, 0, 0, 0, time.UTC)
			results = append(results, chart.DataPoint{
				Timestamp: ts,
				Value:     count,
			})
		}
	}

	return results, totalHosts, nil
}

func (s *Service) CleanupData(ctx context.Context, days int) error {
	return s.store.CleanupBlobData(ctx, days)
}

// fillZeroValues fills in missing time buckets with zero values.
func fillZeroValues(data []chart.DataPoint, startDate, endDate time.Time, downsample int) []chart.DataPoint {
	existing := make(map[time.Time]int, len(data))
	for _, dp := range data {
		existing[dp.Timestamp] = dp.Value
	}

	step := 1
	if downsample > 0 {
		step = downsample
	}

	// Align end to the current step-aligned hour, then walk back from end to
	// produce exactly the right number of data points ending at the current hour.
	endHour := endDate.Hour()
	if step > 1 {
		endHour = (endHour / step) * step
	}
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), endHour, 0, 0, 0, time.UTC)

	// The inclusive loop produces numStepsBack+1 data points. For days=1 hourly
	// we want 24 points, so step back 23 times from end.
	totalHours := int(endDate.Sub(startDate).Hours())
	numStepsBack := totalHours/step - 1
	start := end.Add(-time.Duration(numStepsBack) * time.Duration(step) * time.Hour)

	var result []chart.DataPoint
	for t := start; !t.After(end); t = t.Add(time.Duration(step) * time.Hour) {
		val := existing[t]
		result = append(result, chart.DataPoint{
			Timestamp: t,
			Value:     val,
		})
	}
	return result
}
