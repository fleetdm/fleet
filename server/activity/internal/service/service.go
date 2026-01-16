// Package service provides the service implementation for the activity bounded context.
package service

import (
	"context"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	kitlog "github.com/go-kit/log"
)

// Service is the activity bounded context service implementation.
type Service struct {
	authz  platform_authz.Authorizer
	store  types.Datastore
	users  activity.UserProvider
	logger kitlog.Logger
}

// NewService creates a new activity service.
func NewService(authz platform_authz.Authorizer, store types.Datastore, users activity.UserProvider, logger kitlog.Logger) *Service {
	return &Service{
		authz:  authz,
		store:  store,
		users:  users,
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
	if err := s.authz.Authorize(ctx, &api.Activity{}, actionRead); err != nil {
		return nil, nil, err
	}

	// If searching, also search users table to get matching user IDs
	if opt.MatchQuery != "" {
		userIDs, err := s.users.FindUserIDs(ctx, opt.MatchQuery)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "failed to search users for activity query")
		}
		internalOpt.MatchingUserIDs = userIDs
	}

	activities, meta, err := s.store.ListActivities(ctx, internalOpt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list activities")
	}

	// Enrich activities with user data via ACL
	if err := s.enrichWithUserData(ctx, activities); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "failed to enrich activities with user data")
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

// actionRead is the authorization action for reading activities.
const actionRead = "read"
