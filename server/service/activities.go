package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get activities
////////////////////////////////////////////////////////////////////////////////

type listActivitiesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listActivitiesResponse struct {
	Meta       *fleet.PaginationMetadata `json:"meta"`
	Activities []*fleet.Activity         `json:"activities"`
	Err        error                     `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listActivitiesRequest)
	activities, metadata, err := svc.ListActivities(ctx, fleet.ListActivitiesOptions{
		ListOptions: req.ListOptions,
	})
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return listActivitiesResponse{Meta: metadata, Activities: activities}, nil
}

// ListActivities returns a slice of activities for the whole organization
func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}
	return svc.ds.ListActivities(ctx, opt)
}

func (svc *Service) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	return svc.ds.NewActivity(ctx, user, activity)
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
	Activities []*fleet.Activity         `json:"activities"`
	Count      uint                      `json:"count"`
	Err        error                     `json:"error,omitempty"`
}

func (r listHostUpcomingActivitiesResponse) error() error { return r.Err }

func listHostUpcomingActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostUpcomingActivitiesRequest)
	acts, meta, err := svc.ListHostUpcomingActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listHostUpcomingActivitiesResponse{Err: err}, nil
	}

	return listHostUpcomingActivitiesResponse{Meta: meta, Activities: acts, Count: meta.TotalResults}, nil
}

// ListHostUpcomingActivities returns a slice of upcoming activities for the
// specified host.
func (svc *Service) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
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
	// custom ordering is not supported, always by date (oldest first)
	opt.OrderKey = "created_at"
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata
	opt.IncludeMetadata = true

	return svc.ds.ListHostUpcomingActivities(ctx, hostID, opt)
}

////////////////////////////////////////////////////////////////////////////////
// List host past activities
////////////////////////////////////////////////////////////////////////////////

type listHostPastActivitiesRequest struct {
	HostID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

func listHostPastActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostPastActivitiesRequest)
	acts, meta, err := svc.ListHostPastActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return &listActivitiesResponse{Meta: meta, Activities: acts}, nil
}

func (svc *Service) ListHostPastActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
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

	// cursor-based pagination is not supported for past activities
	opt.After = ""
	// custom ordering is not supported, always by date (newest first)
	opt.OrderKey = "created_at"
	opt.OrderDirection = fleet.OrderDescending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata
	opt.IncludeMetadata = true

	return svc.ds.ListHostPastActivities(ctx, hostID, opt)
}
