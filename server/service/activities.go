package service

import (
	"context"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Activities response (used by host past activities endpoint)
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) GetActivitiesWebhookSettings(ctx context.Context) (fleet.ActivitiesWebhookSettings, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return fleet.ActivitiesWebhookSettings{}, ctxerr.Wrap(ctx, err, "get app config for activities webhook")
	}
	return appConfig.WebhookSettings.ActivitiesWebhook, nil
}

func (svc *Service) ActivateNextUpcomingActivityForHost(ctx context.Context, hostID uint, fromCompletedExecID string) error {
	return svc.ds.ActivateNextUpcomingActivityForHost(ctx, hostID, fromCompletedExecID)
}

func (svc *Service) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	var apiUser *activity_api.User
	if user != nil {
		apiUser = &activity_api.User{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Deleted: user.Deleted,
		}
	}
	return svc.activitySvc.NewActivity(ctx, apiUser, activity)
}

// List host upcoming activities
func listHostUpcomingActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListHostUpcomingActivitiesRequest)
	acts, meta, err := svc.ListHostUpcomingActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return fleet.ListHostUpcomingActivitiesResponse{Err: err}, nil
	}

	return fleet.ListHostUpcomingActivitiesResponse{Meta: meta, Activities: acts, Count: meta.TotalResults}, nil
}

// ListHostUpcomingActivities returns a slice of upcoming activities for the
// specified host.
func (svc *Service) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.UpcomingActivity, *fleet.PaginationMetadata, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, nil, err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host")
	}
	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported for upcoming activities
	opt.After = ""
	// custom ordering is not supported, always by upcoming queue order
	// (acual order is in the query, not set via ListOptions)
	opt.OrderKey = ""
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata
	opt.IncludeMetadata = true

	return svc.ds.ListHostUpcomingActivities(ctx, hostID, opt)
}

// List host past activities
// Cancel host upcoming activity
func cancelHostUpcomingActivityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CancelHostUpcomingActivityRequest)
	err := svc.CancelHostUpcomingActivity(ctx, req.HostID, req.ActivityID)
	if err != nil {
		return fleet.CancelHostUpcomingActivityResponse{Err: err}, nil
	}
	return fleet.CancelHostUpcomingActivityResponse{}, nil
}

func (svc *Service) CancelHostUpcomingActivity(ctx context.Context, hostID uint, executionID string) error {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host")
	}
	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionCancelHostActivity); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	// prevent cancellation of lock/wipe that are already activated
	actMeta, err := svc.ds.GetHostUpcomingActivityMeta(ctx, hostID, executionID)
	if err != nil {
		return err
	}
	if actMeta.ActivatedAt != nil &&
		(actMeta.WellKnownAction == fleet.WellKnownActionLock || actMeta.WellKnownAction == fleet.WellKnownActionWipe) {
		return &fleet.BadRequestError{
			Message: "Couldn't cancel activity. Lock and wipe can't be canceled if they're about to run to prevent you from losing access to the host.",
		}
	}

	pastAct, err := svc.ds.CancelHostUpcomingActivity(ctx, hostID, executionID)
	if err != nil {
		return err
	}

	if pastAct != nil {
		if err := svc.NewActivity(ctx, vc.User, pastAct); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for cancelation")
		}
	}
	return nil
}
