// Package activityacl provides the anti-corruption layer between the activity
// bounded context and legacy Fleet code.
//
// This package is the ONLY place that imports both activity types and fleet types.
// It translates between them, allowing the activity context to remain decoupled
// from legacy code.
package activityacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// FleetServiceAdapter provides access to Fleet service methods
// for data that the activity bounded context doesn't own.
type FleetServiceAdapter struct {
	svc fleet.ActivityLookupService
}

// NewFleetServiceAdapter creates a new adapter for the Fleet service.
func NewFleetServiceAdapter(svc fleet.ActivityLookupService) *FleetServiceAdapter {
	return &FleetServiceAdapter{svc: svc}
}

// Ensure FleetServiceAdapter implements the required interfaces
var (
	_ activity.UserProvider              = (*FleetServiceAdapter)(nil)
	_ activity.HostProvider              = (*FleetServiceAdapter)(nil)
	_ activity.AppConfigProvider         = (*FleetServiceAdapter)(nil)
	_ activity.UpcomingActivityActivator = (*FleetServiceAdapter)(nil)
	_ activity.WebhookSender             = (*FleetServiceAdapter)(nil)
	_ activity.URLMasker                 = (*FleetServiceAdapter)(nil)
)

// UsersByIDs fetches users by their IDs from the Fleet service.
func (a *FleetServiceAdapter) UsersByIDs(ctx context.Context, ids []uint) ([]*activity.User, error) {
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

// FindUserIDs searches for users by name/email prefix and returns their IDs.
func (a *FleetServiceAdapter) FindUserIDs(ctx context.Context, query string) ([]uint, error) {
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

// GetHostLite fetches minimal host information for authorization.
func (a *FleetServiceAdapter) GetHostLite(ctx context.Context, hostID uint) (*activity.Host, error) {
	host, err := a.svc.GetHostLite(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return &activity.Host{
		ID:     host.ID,
		TeamID: host.TeamID,
	}, nil
}

func convertUser(u *fleet.UserSummary) *activity.User {
	return &activity.User{
		ID:       u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Gravatar: u.GravatarURL,
		APIOnly:  u.APIOnly,
	}
}

// GetActivitiesWebhookConfig returns the webhook configuration for activities.
func (a *FleetServiceAdapter) GetActivitiesWebhookConfig(ctx context.Context) (*activity.ActivitiesWebhookSettings, error) {
	settings, err := a.svc.GetActivitiesWebhookSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &activity.ActivitiesWebhookSettings{
		Enable:         settings.Enable,
		DestinationURL: settings.DestinationURL,
	}, nil
}

// ActivateNextUpcomingActivity activates the next upcoming activity in the queue.
func (a *FleetServiceAdapter) ActivateNextUpcomingActivity(ctx context.Context, hostID uint, fromCompletedExecID string) error {
	return a.svc.ActivateNextUpcomingActivityForHost(ctx, hostID, fromCompletedExecID)
}

// SendWebhookPayload sends a JSON payload to the given URL using the server's HTTP utility.
func (a *FleetServiceAdapter) SendWebhookPayload(ctx context.Context, url string, payload any) error {
	return server.PostJSONWithTimeout(ctx, url, payload)
}

// MaskSecretURLParams masks sensitive parameters in a URL for safe logging.
func (a *FleetServiceAdapter) MaskSecretURLParams(rawURL string) string {
	return server.MaskSecretURLParams(rawURL)
}

// MaskURLError masks sensitive URL information in an error for safe logging.
func (a *FleetServiceAdapter) MaskURLError(err error) error {
	return server.MaskURLError(err)
}
