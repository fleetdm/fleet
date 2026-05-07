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
	viewer    api.ViewerProvider
	datasets  map[string]api.Dataset
	hostCache *hostFilterCache
	logger    *slog.Logger
}

// NewService creates a new chart service.
func NewService(authz platform_authz.Authorizer, store types.Datastore, viewerProvider api.ViewerProvider, logger *slog.Logger) *Service {
	return &Service{
		authz:     authz,
		store:     store,
		viewer:    viewerProvider,
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

func (s *Service) CollectDatasets(ctx context.Context, now time.Time, scope api.CollectScopeFn) error {
	for name, dataset := range s.datasets {
		var disabledFleetIDs []uint
		if scope != nil {
			skip, disabled := scope(name)
			if skip {
				continue
			}
			disabledFleetIDs = disabled
		}
		if err := dataset.Collect(ctx, s.store, now, disabledFleetIDs); err != nil {
			// Log and continue — don't let one dataset failure block others.
			if s.logger != nil {
				s.logger.ErrorContext(ctx, "collect chart dataset", "dataset", name, "err", ctxerr.Wrap(ctx, err, "collect chart dataset"))
			}
		}
	}
	return nil
}

func (s *Service) GetChartData(ctx context.Context, metric string, opts api.RequestOpts) (*api.Response, error) {
	// Resolve scope first: for authz we need the right action + subject, and
	// for data we need the effective team set. Fail closed if there's no
	// viewer — the authenticated middleware should have placed one in ctx.
	isGlobal, viewerTeamIDs, err := s.viewer.ViewerScope(ctx)
	if err != nil {
		return nil, err
	}

	// Build the authz subject + action. Two distinct cases:
	//   - Explicit team_id: Host{TeamID: opts.TeamID} + ActionRead. Rego's
	//     read rule for hosts requires team_role(subject, object.team_id) to
	//     match, so a team user asking for a team they don't have a role on
	//     is rejected by policy (not by us). Global users pass via the
	//     global-role rules, which don't care about team_id.
	//   - No team_id: Host{} + ActionList. Rego's list rules pass global
	//     users unconditionally and pass team users who have a list-capable
	//     role on any of their teams. The service then scopes data below.
	authzSubject := &api.Host{TeamID: opts.TeamID}
	authzAction := platform_authz.ActionRead
	if opts.TeamID == nil {
		authzAction = platform_authz.ActionList
	}
	if err := s.authz.Authorize(ctx, authzSubject, authzAction); err != nil {
		return nil, err
	}

	dataset, ok := s.datasets[metric]
	if !ok {
		return nil, &platform_http.BadRequestError{Message: fmt.Sprintf("unknown chart metric: %s", metric)}
	}

	// Resolution must be 0 or a positive divisor of 24.
	if opts.Resolution < 0 || (opts.Resolution != 0 && 24%opts.Resolution != 0) {
		return nil, &platform_http.BadRequestError{Message: fmt.Sprintf("invalid resolution value: %d (must be 0 or a positive divisor of 24)", opts.Resolution)}
	}

	hours := opts.Resolution
	if hours <= 0 {
		hours = dataset.DefaultResolutionHours()
	}
	bucketSize := time.Duration(hours) * time.Hour

	startDate, endDate := computeBucketRange(time.Now(), bucketSize, opts.Days, opts.TZOffsetMinutes)

	// Build the host filter. The bitmap mask always encodes "currently visible
	// hosts" — team scoping, label/platform/include/exclude, and incidentally
	// dropping hosts deleted since the SCD rows were written.
	hostFilter := &types.HostFilter{
		TeamIDs:        effectiveTeamIDs(opts.TeamID, isGlobal, viewerTeamIDs),
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

	var entityIDs []string
	if metric == "cve" {
		// TODO(iteration-2): replace with user-configurable filter from
		// RequestOpts when dynamic CVE filtering ships.
		entityIDs, err = s.store.TrackedCriticalCVEs(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "resolve tracked critical CVEs")
		}
	}

	// entityIDs semantics at the storage layer: nil = no filter; non-nil empty
	// = match nothing (produces zero-valued buckets). Do NOT convert empty to
	// nil here.
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

// effectiveTeamIDs decides the team scope applied at SQL time.
//
//	explicit team_id? → just that team (authz rule above already ensured
//	                    the caller has access to it, or is global)
//	global user, no team_id → nil, meaning "no team filter"
//	team user, no team_id   → the viewer's accessible teams. Empty-but-non-nil
//	                          here means the user has no teams at all; SQL
//	                          emits 1=0 so they see nothing.
func effectiveTeamIDs(requestedTeamID *uint, isGlobal bool, viewerTeamIDs []uint) []uint {
	if requestedTeamID != nil {
		return []uint{*requestedTeamID}
	}
	if isGlobal {
		return nil
	}
	// Return a non-nil slice even when empty — the SQL builder treats non-nil
	// as "scoped" and emits a no-match clause, which is what we want for a
	// team user with zero team memberships.
	if viewerTeamIDs == nil {
		return []uint{}
	}
	return viewerTeamIDs
}

func (s *Service) CleanupData(ctx context.Context, days int) error {
	return s.store.CleanupSCDData(ctx, days)
}

// scrubBatchSize is the upper bound on rows touched per statement during
// scrub jobs. Tunable; chosen to bound lock duration without producing
// excessive round trips on large tables.
const scrubBatchSize = 5000

func (s *Service) ScrubDatasetGlobal(ctx context.Context, dataset string) error {
	return s.store.DeleteAllForDataset(ctx, dataset, scrubBatchSize)
}

func (s *Service) ScrubDatasetFleet(ctx context.Context, dataset string, fleetIDs []uint) error {
	if len(fleetIDs) == 0 {
		// Defensive: an enqueue with an empty fleet list shouldn't happen, but
		// if it does treat as a no-op rather than scanning the entire table.
		if s.logger != nil {
			s.logger.WarnContext(ctx, "chart fleet scrub invoked with empty fleet list", "dataset", dataset)
		}
		return nil
	}
	hostIDs, err := s.store.HostIDsInFleets(ctx, fleetIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load hosts for chart fleet scrub")
	}
	if len(hostIDs) == 0 {
		// No hosts currently in any of those fleets — fleets may have been
		// deleted or every host moved out before the scrub ran. Best-effort
		// per design decision; no work to do.
		if s.logger != nil {
			s.logger.InfoContext(ctx, "chart fleet scrub: no hosts resolved", "dataset", dataset, "fleet_ids", fleetIDs)
		}
		return nil
	}
	mask := chart.HostIDsToBlob(hostIDs)
	return s.store.ApplyScrubMaskToDataset(ctx, dataset, mask, scrubBatchSize)
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
