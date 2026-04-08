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

	data, err := s.store.GetChartData(ctx, metric, startDate, endDate, hostFilter, entityIDs, dataset.HasEntityDimension(), opts.Downsample)
	if err != nil {
		return nil, err
	}

	totalHosts, err := s.store.CountHostsForChartFilter(ctx, hostFilter)
	if err != nil {
		return nil, err
	}

	resolution := "hourly"
	if opts.Downsample > 0 {
		resolution = fmt.Sprintf("%d-hour", opts.Downsample)
	}

	data = fillZeroValues(data, startDate, endDate, opts.Downsample)

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

func (s *Service) CleanupData(ctx context.Context, days int) error {
	return s.store.CleanupHostHourlyData(ctx, days)
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

	// Align start hour to the step boundary so timestamps match the SQL output
	// (which always uses step-aligned hours from midnight: 0, 2, 4... or 0, 4, 8...).
	startHour := startDate.Hour()
	if step > 1 {
		startHour = (startHour / step) * step
	}
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), startHour, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), endDate.Hour(), 0, 0, 0, time.UTC)

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
