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
	userSvc fleet.UserLookupService
	hostSvc fleet.HostLookupService
}

// NewFleetServiceAdapter creates a new adapter for the Fleet service.
func NewFleetServiceAdapter(userSvc fleet.UserLookupService, hostSvc fleet.HostLookupService) *FleetServiceAdapter {
	return &FleetServiceAdapter{userSvc: userSvc, hostSvc: hostSvc}
}

// Ensure FleetServiceAdapter implements activity.UserProvider and activity.HostProvider
var (
	_ activity.UserProvider = (*FleetServiceAdapter)(nil)
	_ activity.HostProvider = (*FleetServiceAdapter)(nil)
)

// UsersByIDs fetches users by their IDs from the Fleet service.
func (a *FleetServiceAdapter) UsersByIDs(ctx context.Context, ids []uint) ([]*activity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Fetch only the requested users by their IDs
	users, err := a.userSvc.UsersByIDs(ctx, ids)
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

// FindUserIDs searches for users by name/email prefix and returns their IDs.
func (a *FleetServiceAdapter) FindUserIDs(ctx context.Context, query string) ([]uint, error) {
	if query == "" {
		return nil, nil
	}

	// Search users via Fleet service with the query
	users, err := a.userSvc.ListUsers(ctx, fleet.UserListOptions{
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

// GetHostLite fetches minimal host information for authorization.
func (a *FleetServiceAdapter) GetHostLite(ctx context.Context, hostID uint) (*activity.Host, error) {
	host, err := a.hostSvc.GetHostLite(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return &activity.Host{
		ID:     host.ID,
		TeamID: host.TeamID,
	}, nil
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
