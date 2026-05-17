package service

import (
	"context"
	"net/http"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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

func (svc *Service) NewActivity(ctx context.Context, user *fleet.User, activity activity_api.ActivityDetails) error {
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

////////////////////////////////////////////////////////////////////////////////
// List host upcoming activities
////////////////////////////////////////////////////////////////////////////////

type listHostUpcomingActivitiesRequest struct {
	HostID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listHostUpcomingActivitiesResponse struct {
	Meta       *fleet.PaginationMetadata `json:"meta"`
	Activities []*fleet.UpcomingActivity `json:"activities"`
	Count      uint                      `json:"count"`
	Err        error                     `json:"error,omitempty"`
}

func (r listHostUpcomingActivitiesResponse) Error() error { return r.Err }

func listHostUpcomingActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostUpcomingActivitiesRequest)
	acts, meta, err := svc.ListHostUpcomingActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listHostUpcomingActivitiesResponse{Err: err}, nil
	}

	return listHostUpcomingActivitiesResponse{Meta: meta, Activities: acts, Count: meta.TotalResults}, nil
}

// ListHostUpcomingActivities returns a slice of upcoming activities for the
// specified host.
func (svc *Service) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.UpcomingActivity, *fleet.PaginationMetadata, error) {
	// Device-authenticated callers (e.g. the My Device page) have already had
	// host access verified by the device-token middleware. Skip user-mode authz
	// in that case and use the host injected into context. Require the caller
	// to pass the matching host ID so that no caller can ever read another
	// host's queue under a device token.
	if svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) ||
		svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceCertificate) ||
		svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceURL) {
		h, ok := hostctx.FromContext(ctx)
		if !ok {
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		}
		if h.ID != hostID {
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device authentication does not match requested host"))
		}
	} else {
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

// ListHostPastActivitiesForDevice returns past activities for the specified
// host in a device-token-authenticated context. The device-token middleware is
// expected to have already established access to the host; no further
// authorization is performed here beyond delegating to the activity bounded
// context's device variant.
func (svc *Service) ListHostPastActivitiesForDevice(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) &&
		!svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceCertificate) &&
		!svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceURL) {
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device authentication required"))
	}
	h, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}
	if h.ID != hostID {
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device token does not match requested host"))
	}

	apiDirection := activity_api.OrderAscending
	if opt.OrderDirection == fleet.OrderDescending {
		apiDirection = activity_api.OrderDescending
	}
	apiOpt := activity_api.ListOptions{
		Page:           opt.Page,
		PerPage:        opt.PerPage,
		After:          opt.After,
		OrderKey:       opt.OrderKey,
		OrderDirection: apiDirection,
		MatchQuery:     opt.MatchQuery,
	}

	acts, apiMeta, err := svc.activitySvc.ListHostPastActivitiesForDevice(ctx, hostID, apiOpt)
	if err != nil {
		return nil, nil, err
	}
	var meta *fleet.PaginationMetadata
	if apiMeta != nil {
		meta = &fleet.PaginationMetadata{
			HasNextResults:     apiMeta.HasNextResults,
			HasPreviousResults: apiMeta.HasPreviousResults,
			TotalResults:       apiMeta.TotalResults,
		}
	}
	return acts, meta, nil
}

////////////////////////////////////////////////////////////////////////////////
// Cancel host upcoming activity
////////////////////////////////////////////////////////////////////////////////

type cancelHostUpcomingActivityRequest struct {
	HostID     uint   `url:"id"`
	ActivityID string `url:"activity_id"`
}

type cancelHostUpcomingActivityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r cancelHostUpcomingActivityResponse) Error() error { return r.Err }
func (r cancelHostUpcomingActivityResponse) Status() int  { return http.StatusNoContent }

func cancelHostUpcomingActivityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*cancelHostUpcomingActivityRequest)
	err := svc.CancelHostUpcomingActivity(ctx, req.HostID, req.ActivityID)
	if err != nil {
		return cancelHostUpcomingActivityResponse{Err: err}, nil
	}
	return cancelHostUpcomingActivityResponse{}, nil
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
