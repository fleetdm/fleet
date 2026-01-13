// Package activityacl provides the anti-corruption layer between the activity
// bounded context and legacy Fleet code.
//
// This package is the ONLY place that imports both activity types and fleet types.
// It translates between them, allowing the activity context to remain decoupled
// from legacy code.
package activityacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// FleetServiceAdapter provides access to Fleet service methods
// for data that the activity bounded context doesn't own.
type FleetServiceAdapter struct {
	svc fleet.Service
}

// NewFleetServiceAdapter creates a new adapter for the Fleet service.
func NewFleetServiceAdapter(svc fleet.Service) *FleetServiceAdapter {
	return &FleetServiceAdapter{svc: svc}
}

// Ensure FleetServiceAdapter implements activity.UserProvider
var _ activity.UserProvider = (*FleetServiceAdapter)(nil)

// ListUsers fetches users by their IDs from the Fleet service.
func (a *FleetServiceAdapter) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Fetch only the requested users by their IDs
	users, err := a.svc.UsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Convert to activity.User
	result := make([]*activity.User, 0, len(users))
	for _, u := range users {
		result = append(result, convertUser(u))
	}
	return result, nil
}

// SearchUsers searches for users by name/email prefix and returns their IDs.
func (a *FleetServiceAdapter) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	if query == "" {
		return nil, nil
	}

	// Search users via Fleet service with the query
	users, err := a.svc.ListUsers(ctx, fleet.UserListOptions{
		ListOptions: fleet.ListOptions{
			MatchQuery: query,
		},
	})
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	return ids, nil
}

func convertUser(u *fleet.User) *activity.User {
	return &activity.User{
		ID:       u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Gravatar: u.GravatarURL,
		APIOnly:  u.APIOnly,
	}
}
