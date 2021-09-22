package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listSoftwareRequest struct {
	TeamID      *uint             `query:"team_id,optional"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listSoftwareResponse struct {
	Software []fleet.Software `json:"software,omitempty"`
	Err      error            `json:"error,omitempty"`
}

func (r listSoftwareResponse) error() error { return r.Err }

func listSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listSoftwareRequest)
	resp, err := svc.ListSoftware(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listSoftwareResponse{Err: err}, nil
	}
	return listSoftwareResponse{Software: resp}, nil
}

func (svc Service) ListSoftware(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Software{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListSoftware(ctx, teamID, opt)
}
