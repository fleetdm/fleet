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

// LegacyServiceAdapter provides access to legacy service methods
// for data that the activity bounded context doesn't own.
type LegacyServiceAdapter struct {
	svc fleet.Service
}

// NewLegacyServiceAdapter creates a new adapter for the legacy service.
func NewLegacyServiceAdapter(svc fleet.Service) *LegacyServiceAdapter {
	return &LegacyServiceAdapter{svc: svc}
}

// Ensure LegacyServiceAdapter implements activity.UserProvider
var _ activity.UserProvider = (*LegacyServiceAdapter)(nil)

// ListUsers fetches users by their IDs from the legacy service.
func (a *LegacyServiceAdapter) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build a set for quick lookup
	idSet := make(map[uint]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	// Fetch all users from legacy service
	// TODO: This is inefficient - ideally we'd have a method to fetch by IDs
	users, err := a.svc.ListUsers(ctx, fleet.UserListOptions{})
	if err != nil {
		return nil, err
	}

	// Filter to requested IDs and convert
	result := make([]*activity.User, 0, len(ids))
	for _, u := range users {
		if idSet[u.ID] {
			result = append(result, convertUser(u))
		}
	}
	return result, nil
}

// SearchUsers searches for users by name/email prefix and returns their IDs.
// This matches the legacy behavior in server/datastore/mysql/activities.go ListActivities
// where it searches users to get the most up-to-date name/email info.
func (a *LegacyServiceAdapter) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	if query == "" {
		return nil, nil
	}

	// Search users via legacy service with the query
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
	gravatar := ""
	if u.GravatarURL != "" {
		gravatar = u.GravatarURL
	}
	return &activity.User{
		ID:       u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Gravatar: gravatar,
		APIOnly:  u.APIOnly,
	}
}
