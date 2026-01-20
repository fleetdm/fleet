// Package service provides the service implementation for the activity bounded context.
package service

import (
	"context"
	"encoding/json"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/go-multierror"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz  platform_authz.Authorizer
	store  types.Datastore
	users  activity.UserProvider
	hosts  activity.HostProvider
	logger kitlog.Logger
}

// NewService creates a new activity service.
func NewService(authz platform_authz.Authorizer, store types.Datastore, users activity.UserProvider, hosts activity.HostProvider, logger kitlog.Logger) *Service {
	return &Service{
		authz:  authz,
		store:  store,
		users:  users,
		hosts:  hosts,
		logger: logger,
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

// ListActivities returns a slice of activities for the whole organization.
func (s *Service) ListActivities(ctx context.Context, opt api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	// Convert public options to internal options (which include internal fields)
	// Don't include metadata for cursor-based pagination (when After is set)
	internalOpt := types.ListOptions{
		ListOptions:     opt,
		IncludeMetadata: opt.After == "",
	}

	// Authorization check
	if err := s.authz.Authorize(ctx, &api.Activity{}, platform_authz.ActionRead); err != nil {
		return nil, nil, err
	}

	// If searching, also search users table to get matching user IDs.
	// Use graceful degradation for authorization errors only: if user search fails
	// due to authorization, proceed without user-based filtering rather than failing.
	if opt.MatchQuery != "" {
		userIDs, err := s.users.FindUserIDs(ctx, opt.MatchQuery)
		switch {
		case err == nil:
			internalOpt.MatchingUserIDs = userIDs
		case platform_authz.IsForbidden(err):
			level.Debug(s.logger).Log("msg", "user search forbidden, proceeding without user filter", "err", err)
		default:
			return nil, nil, ctxerr.Wrap(ctx, err, "failed to search users for activity query")
		}
	}

	activities, meta, err := s.store.ListActivities(ctx, internalOpt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list activities")
	}

	// Enrich activities with user data via ACL.
	// Use graceful degradation for authorization errors only: if user enrichment fails
	// due to authorization, return activities without user data rather than failing.
	if err := s.enrichWithUserData(ctx, activities); err != nil {
		if platform_authz.IsForbidden(err) {
			level.Debug(s.logger).Log("msg", "user enrichment forbidden, proceeding without enrichment", "err", err)
		} else {
			return nil, nil, ctxerr.Wrap(ctx, err, "failed to enrich activities with user data")
		}
	}

	return activities, meta, nil
}

// MarkActivitiesAsStreamed marks the given activities as streamed.
// This is called by the cron job after successfully streaming activities to the audit logger.
// No authorization required as this is an internal operation.
func (s *Service) MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error {
	return s.store.MarkActivitiesAsStreamed(ctx, activityIDs)
}

// ListHostPastActivities returns past activities for a specific host.
func (s *Service) ListHostPastActivities(ctx context.Context, hostID uint, opt api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	// First ensure the user has access to list hosts
	if err := s.authz.Authorize(ctx, &api.Activity{}, platform_authz.ActionList); err != nil {
		return nil, nil, err
	}

	// Fetch host to get team_id for authorization
	host, err := s.hosts.GetHostLite(ctx, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host")
	}

	// Authorize again with team loaded now that we have team_id
	if err := s.authz.Authorize(ctx, host, platform_authz.ActionRead); err != nil {
		return nil, nil, err
	}

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
		if platform_authz.IsForbidden(err) {
			level.Debug(s.logger).Log("msg", "user enrichment forbidden, proceeding without enrichment", "err", err)
		} else {
			return nil, nil, ctxerr.Wrap(ctx, err, "failed to enrich activities with user data")
		}
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

	users, err := s.users.UsersByIDs(ctx, slices.Collect(maps.Keys(lookup)))
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
// batchSize controls how many activities are fetched per batch.
func (s *Service) StreamActivities(systemCtx context.Context, auditLogger api.JSONLogger, batchSize uint) error {
	page := uint(0)
	for {
		// (1) Get batch of activities that haven't been streamed.
		activitiesToStream, _, err := s.ListActivities(systemCtx, api.ListOptions{
			OrderKey:       "id",
			OrderDirection: api.OrderAscending,
			PerPage:        batchSize,
			Page:           page,
			Streamed:       ptrBool(false),
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
		for _, activity := range activitiesToStream {
			b, err := json.Marshal(activity)
			if err != nil {
				return ctxerr.Wrap(systemCtx, err, "marshal activity")
			}
			if err := auditLogger.Write(systemCtx, []json.RawMessage{json.RawMessage(b)}); err != nil {
				if len(streamedIDs) == 0 {
					return ctxerr.Wrapf(systemCtx, err, "stream first activity: %d", activity.ID)
				}
				multiErr = multierror.Append(multiErr, ctxerr.Wrapf(systemCtx, err, "stream activity: %d", activity.ID))
				// We stop streaming upon the first error (will retry on next cron iteration)
				break
			}
			streamedIDs = append(streamedIDs, activity.ID)
		}

		s.logger.Log("streamed-events", len(streamedIDs))

		// (3) Mark the streamed activities as streamed.
		if err := s.MarkActivitiesAsStreamed(systemCtx, streamedIDs); err != nil {
			multiErr = multierror.Append(multiErr, ctxerr.Wrap(systemCtx, err, "mark activities as streamed"))
		}

		// If there was an error while streaming or updating activities, return.
		if multiErr != nil {
			return multiErr
		}

		if len(activitiesToStream) < int(batchSize) { //nolint:gosec // dismiss G115
			return nil
		}
		page++
	}
}

// ptrBool returns a pointer to the given bool value.
func ptrBool(b bool) *bool {
	return &b
}
