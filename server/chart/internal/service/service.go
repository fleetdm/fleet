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
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// Service is the chart bounded context service implementation.
type Service struct {
	authz     platform_authz.Authorizer
	store     types.Datastore
	datasets  map[string]api.Dataset
	hostCache *hostFilterCache
	logger    *slog.Logger
}

// NewService creates a new chart service.
func NewService(authz platform_authz.Authorizer, store types.Datastore, logger *slog.Logger) *Service {
	return &Service{
		authz:     authz,
		store:     store,
		datasets:  make(map[string]api.Dataset),
		hostCache: newHostFilterCache(hostFilterCacheTTL),
		logger:    logger,
	}
}

// Ensure Service implements api.Service at compile time.
var _ api.Service = (*Service)(nil)

func (s *Service) RegisterDataset(ds api.Dataset) {
	s.datasets[ds.Name()] = ds
}

func (s *Service) CollectDatasets(ctx context.Context, now time.Time) error {
	for name, dataset := range s.datasets {
		if err := dataset.Collect(ctx, s.store, now); err != nil {
			// Log and continue — don't let one dataset failure block others.
			s.logger.ErrorContext(ctx, "collect chart dataset", "dataset", name, "err", ctxerr.Wrap(ctx, err, "collect chart dataset"))
		}
	}
	return nil
}

func (s *Service) GetChartData(ctx context.Context, metric string, opts api.RequestOpts) (*api.Response, error) {
	// Authorize against a Host carrying the requested team scope. With opts.TeamID
	// unset, rego's team rules can't match (object.team_id undefined), so only
	// global-role users pass — a team observer must specify their team_id.
	if err := s.authz.Authorize(ctx, &api.Host{TeamID: opts.TeamID}, platform_authz.ActionRead); err != nil {
		return nil, err
	}

	dataset, ok := s.datasets[metric]
	if !ok {
		return nil, &platform_http.BadRequestError{Message: fmt.Sprintf("unknown chart metric: %s", metric)}
	}

	// Validate days preset.
	validDays := map[int]struct{}{1: {}, 7: {}, 14: {}, 30: {}}
	if _, ok := validDays[opts.Days]; !ok {
		return nil, &platform_http.BadRequestError{Message: fmt.Sprintf("invalid days value: %d (must be 1, 7, 14, or 30)", opts.Days)}
	}

	// Downsample only makes sense for hourly datasets and must be 0 or a positive divisor of 24.
	if opts.Downsample < 0 || (opts.Downsample != 0 && 24%opts.Downsample != 0) {
		return nil, &platform_http.BadRequestError{Message: fmt.Sprintf("invalid downsample value: %d (must be 0 or a positive divisor of 24)", opts.Downsample)}
	}

	// Resolve dataset-specific filters to entity IDs.
	entityIDs, err := dataset.ResolveFilters(ctx, s.store, opts.DatasetFilters)
	if err != nil {
		return nil, err
	}

	// Compute effective bucket size. Downsample applies only to sub-daily datasets.
	bucketSize := dataset.BucketSize()
	if bucketSize < 24*time.Hour && opts.Downsample > 1 {
		bucketSize = time.Duration(opts.Downsample) * time.Hour
	}

	startDate, endDate := computeBucketRange(time.Now(), bucketSize, opts.Days, opts.TZOffsetMinutes)

	// Build the host filter. Always non-nil so the bitmap mask encodes "currently
	// visible hosts" — this both applies team/label scoping and drops hosts that
	// have been deleted since the SCD rows were written.
	hostFilter := &types.HostFilter{
		TeamID:         opts.TeamID,
		LabelIDs:       opts.LabelIDs,
		Platforms:      opts.Platforms,
		IncludeHostIDs: opts.IncludeHostIDs,
		ExcludeHostIDs: opts.ExcludeHostIDs,
	}

	filterMask, err := s.hostCache.Get(ctx, hostFilter, func(ctx context.Context) ([]byte, error) {
		hostIDs, err := s.store.GetHostIDsForFilter(ctx, hostFilter)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fetch host IDs for chart filter")
		}
		return chart.HostIDsToBlob(hostIDs), nil
	})
	if err != nil {
		return nil, err
	}

	data, err := s.store.GetSCDData(ctx, metric, startDate, endDate, bucketSize, dataset.SampleStrategy(), filterMask, entityIDs)
	if err != nil {
		return nil, err
	}

	return &api.Response{
		Metric:        metric,
		Visualization: dataset.DefaultVisualization(),
		TotalHosts:    chart.BlobPopcount(filterMask),
		Resolution:    formatResolution(bucketSize),
		Days:          opts.Days,
		Filters: api.Filters{
			TeamID:         opts.TeamID,
			LabelIDs:       opts.LabelIDs,
			Platforms:      opts.Platforms,
			IncludeHostIDs: opts.IncludeHostIDs,
			ExcludeHostIDs: opts.ExcludeHostIDs,
		},
		Data: data,
	}, nil
}

func (s *Service) CleanupData(ctx context.Context, days int) error {
	return s.store.CleanupSCDData(ctx, days)
}

// computeBucketRange returns a (startDate, endDate) UTC pair such that the
// GetSCDData walker will emit (days*24h)/bucketSize data points labeled at
// bucket boundaries aligned to the client's local time. The last label is
// endDate — i.e., the current (possibly ongoing) bucket in the client's tz.
func computeBucketRange(now time.Time, bucketSize time.Duration, days, tzOffsetMinutes int) (time.Time, time.Time) {
	loc := time.FixedZone("client", -tzOffsetMinutes*60)
	localNow := now.In(loc)

	var alignedEnd time.Time
	if bucketSize < 24*time.Hour {
		// Align to the current local bucket within the day.
		step := max(int(bucketSize/time.Hour), 1)
		alignedHour := (localNow.Hour() / step) * step
		alignedEnd = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), alignedHour, 0, 0, 0, loc)
	} else {
		// Daily (or coarser) — align to the start of today's local day.
		alignedEnd = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	}

	endDate := alignedEnd.UTC()
	startDate := endDate.Add(-time.Duration(days) * 24 * time.Hour)
	return startDate, endDate
}

func formatResolution(bucketSize time.Duration) string {
	switch {
	case bucketSize == time.Hour:
		return "hourly"
	case bucketSize == 24*time.Hour:
		return "daily"
	case bucketSize < 24*time.Hour:
		return fmt.Sprintf("%d-hour", int(bucketSize/time.Hour))
	default:
		return fmt.Sprintf("%d-day", int(bucketSize/(24*time.Hour)))
	}
}
