// Package service provides the service implementation for the activity bounded context.
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/hashicorp/go-multierror"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/activity/internal/service")

// streamBatchSize is the number of activities to fetch per batch when streaming.
const streamBatchSize uint = 500

// applyListOptionsDefaults sets sensible defaults for list options.
// This ensures consistent behavior whether the service is called via HTTP or directly.
func applyListOptionsDefaults(opt *api.ListOptions, defaultOrderKey string) {
	// Default ordering (newest first) if not specified
	if opt.OrderKey == "" {
		opt.OrderKey = defaultOrderKey
		opt.OrderDirection = api.OrderDescending
	}
	// Default PerPage based on whether pagination was requested
	if opt.PerPage == 0 {
		if opt.Page == 0 {
			// No pagination requested - return up to maxPerPage results
			opt.PerPage = maxPerPage
		} else {
			// Page specified without per_page - use sensible default
			opt.PerPage = defaultPerPage
		}
	}
}

// Service is the activity bounded context service implementation.
type Service struct {
	authz     platform_authz.Authorizer
	store     types.Datastore
	providers activity.DataProviders
	logger    *slog.Logger
}

// NewService creates a new activity service.
func NewService(
	authz platform_authz.Authorizer,
	store types.Datastore,
	providers activity.DataProviders,
	logger *slog.Logger,
) *Service {
	return &Service{
		authz:     authz,
		store:     store,
		providers: providers,
		logger:    logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

// ListActivities returns a slice of activities for the whole organization.
func (s *Service) ListActivities(ctx context.Context, opt api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	if err := s.authz.Authorize(ctx, &api.Activity{}, platform_authz.ActionRead); err != nil {
		return nil, nil, err
	}

	applyListOptionsDefaults(&opt, "created_at")
	// Convert public options to internal options (which include internal fields)
	// Don't include metadata for cursor-based pagination (when After is set)
	internalOpt := types.ListOptions{
		ListOptions:     opt,
		IncludeMetadata: opt.After == "",
	}

	// If searching, also search users table to get matching user IDs.
	if opt.MatchQuery != "" {
		userIDs, err := s.providers.FindUserIDs(ctx, opt.MatchQuery)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "failed to search users for activity query")
		}
		internalOpt.MatchingUserIDs = userIDs
	}

	activities, meta, err := s.store.ListActivities(ctx, internalOpt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list activities")
	}

	// Enrich activities with user data via ACL.
	if err := s.enrichWithUserData(ctx, activities); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "failed to enrich activities with user data")
	}

	return activities, meta, nil
}

// ListHostPastActivities returns past activities for a specific host.
func (s *Service) ListHostPastActivities(ctx context.Context, hostID uint, opt api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	// First ensure the user has access to list hosts, then check the specific host once team_id is loaded.
	if err := s.authz.Authorize(ctx, &activity.Host{}, platform_authz.ActionList); err != nil {
		return nil, nil, err
	}

	// Fetch host to get team_id for authorization
	host, err := s.providers.GetHostLite(ctx, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host")
	}

	// Authorize again with team loaded now that we have team_id
	if err := s.authz.Authorize(ctx, host, platform_authz.ActionRead); err != nil {
		return nil, nil, err
	}

	applyListOptionsDefaults(&opt, "a.created_at")
	// Convert public options to internal options
	internalOpt := types.ListOptions{
		ListOptions:     opt,
		IncludeMetadata: true,
	}

	activities, meta, err := s.store.ListHostPastActivities(ctx, hostID, internalOpt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host past activities")
	}

	// Enrich activities with user data via ACL
	if err := s.enrichWithUserData(ctx, activities); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "enrich activities with user data")
	}

	return activities, meta, nil
}

// enrichWithUserData adds user data (gravatar, email, name, api_only) to activities by fetching via ACL.
func (s *Service) enrichWithUserData(ctx context.Context, activities []*api.Activity) error {
	// Collect unique user IDs and build lookup of activity indices per user
	lookup := make(map[uint][]int)
	for idx, a := range activities {
		if a.ActorID != nil {
			lookup[*a.ActorID] = append(lookup[*a.ActorID], idx)
		}
	}

	if len(lookup) == 0 {
		return nil
	}

	users, err := s.providers.UsersByIDs(ctx, slices.Collect(maps.Keys(lookup)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list users for activity enrichment")
	}

	// Enrich activities with user data
	for _, user := range users {
		entries, ok := lookup[user.ID]
		if !ok {
			continue
		}
		for _, idx := range entries {
			activities[idx].ActorEmail = &user.Email
			activities[idx].ActorGravatar = &user.Gravatar
			activities[idx].ActorFullName = &user.Name
			activities[idx].ActorAPIOnly = &user.APIOnly
		}
	}

	return nil
}

// StreamActivities streams unstreamed activities to the provided audit logger.
// The systemCtx should be a context with system-level authorization (no user context).
//
// This function uses cursor-based pagination (using activity ID) instead of offset-based
// pagination to handle two scenarios correctly:
//   - Replication lag: The replica may still show activities as unstreamed after they've been
//     marked as streamed on the primary. Cursor-based pagination skips past already-processed
//     IDs regardless of the replica's streamed status.
//   - Result set changes: As activities are marked as streamed, the result set shrinks.
//     Offset-based pagination would skip items, but cursor-based pagination doesn't.
func (s *Service) StreamActivities(systemCtx context.Context, auditLogger api.JSONLogger) error {
	var afterID uint
	for {
		// (1) Get batch of activities that haven't been streamed, starting after the last processed ID.
		activitiesToStream, _, err := s.ListActivities(systemCtx, api.ListOptions{
			OrderKey:       "id",
			OrderDirection: api.OrderAscending,
			PerPage:        streamBatchSize,
			After:          idCursor(afterID),
			Streamed:       ptr.Bool(false),
		})
		if err != nil {
			return ctxerr.Wrap(systemCtx, err, "list activities")
		}
		if len(activitiesToStream) == 0 {
			return nil
		}

		// (2) Stream the activities.
		var (
			streamedIDs []uint
			multiErr    error
		)
		// We stream one activity at a time (instead of writing them all with
		// one auditLogger.Write call) to know which ones succeeded/failed,
		// and also because this method happens asynchronously,
		// so we don't need real-time performance.
		for _, act := range activitiesToStream {
			b, err := json.Marshal(act)
			if err != nil {
				return ctxerr.Wrap(systemCtx, err, "marshal activity")
			}
			if err := auditLogger.Write(systemCtx, []json.RawMessage{json.RawMessage(b)}); err != nil {
				if len(streamedIDs) == 0 {
					return ctxerr.Wrapf(systemCtx, err, "stream first activity: %d", act.ID)
				}
				multiErr = multierror.Append(multiErr, ctxerr.Wrapf(systemCtx, err, "stream activity: %d", act.ID))
				// We stop streaming upon the first error (will retry on next cron iteration)
				break
			}
			streamedIDs = append(streamedIDs, act.ID)
			afterID = act.ID
		}

		s.logger.InfoContext(systemCtx, "streamed events", "count", len(streamedIDs))

		// (3) Mark the streamed activities as streamed.
		if err := s.store.MarkActivitiesAsStreamed(systemCtx, streamedIDs); err != nil {
			multiErr = multierror.Append(multiErr, ctxerr.Wrap(systemCtx, err, "mark activities as streamed"))
		}

		// If there was an error while streaming or updating activities, return.
		if multiErr != nil {
			return multiErr
		}

		if len(activitiesToStream) < int(streamBatchSize) { //nolint:gosec // dismiss G115
			return nil
		}
	}
}

// idCursor converts an activity ID to a cursor string for pagination.
// Returns empty string for ID 0 (start from beginning).
func idCursor(id uint) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(id), 10)
}
