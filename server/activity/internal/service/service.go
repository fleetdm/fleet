// Package service provides the service implementation for the activity bounded context.
package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

// Ensure Service implements types.Service
var _ types.Service = (*Service)(nil)

// ListActivities returns a slice of activities for the whole organization.
func (s *Service) ListActivities(ctx context.Context, opt types.ListOptions) ([]*types.Activity, *types.PaginationMetadata, error) {
	// Authorization: use authz package with local authorization subject
	if err := s.authz.Authorize(ctx, &activityAuthzSubject{}, actionRead); err != nil {
		return nil, nil, err
	}

	// If searching, also search users table to get matching user IDs
	// (matches legacy behavior for finding activities by updated user name/email)
	if opt.MatchQuery != "" {
		userIDs, err := s.users.SearchUsers(ctx, opt.MatchQuery)
		if err != nil {
			// Log but don't fail - we can still search activity table fields
			level.Debug(s.logger).Log("msg", "failed to search users for activity query", "err", err)
		} else {
			opt.MatchingUserIDs = userIDs
		}
	}

	activities, meta, err := s.store.ListActivities(ctx, opt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list activities")
	}

	// Enrich activities with user data via ACL
	if err := s.enrichWithUserData(ctx, activities); err != nil {
		// Log but don't fail - user data enrichment is optional
		level.Debug(s.logger).Log("msg", "failed to enrich activities with user data", "err", err)
	}

	return activities, meta, nil
}

// enrichWithUserData adds user data (gravatar, email, name, api_only) to activities by fetching via ACL.
// This matches the legacy behavior in server/datastore/mysql/activities.go ListActivities.
func (s *Service) enrichWithUserData(ctx context.Context, activities []*types.Activity) error {
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

	// Fetch users via ACL (calls legacy service)
	ids := make([]uint, 0, len(lookup))
	for id := range lookup {
		ids = append(ids, id)
	}
	users, err := s.users.ListUsers(ctx, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list users for activity enrichment")
	}

	// Enrich activities with user data (matching legacy behavior)
	for _, user := range users {
		entries, ok := lookup[user.ID]
		if !ok {
			continue
		}

		email := user.Email
		gravatar := user.Gravatar
		name := user.Name
		apiOnly := user.APIOnly

		for _, idx := range entries {
			activities[idx].ActorEmail = &email
			activities[idx].ActorGravatar = &gravatar
			activities[idx].ActorFullName = &name
			activities[idx].ActorAPIOnly = &apiOnly
		}
	}

	return nil
}

// Authorization constants and types

// actionRead is the authorization action for reading activities.
const actionRead = "read"

// activityAuthzSubject implements platform_authz.AuthzTyper for activity authorization.
// This allows the activity bounded context to use authorization without
// depending on fleet.Activity.
type activityAuthzSubject struct{}

// AuthzType returns the authorization type for activities.
func (a *activityAuthzSubject) AuthzType() string {
	return "activity"
}
