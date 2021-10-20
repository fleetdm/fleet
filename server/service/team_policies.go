package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type teamPolicyRequest struct {
	TeamID     uint   `url:"team_id"`
	QueryID    uint   `json:"query_id"`
	Resolution string `json:"resolution"`
}

type teamPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r teamPolicyResponse) error() error { return r.Err }

func teamPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*teamPolicyRequest)
	resp, err := svc.NewTeamPolicy(ctx, req.TeamID, req.QueryID, "")
	if err != nil {
		return teamPolicyResponse{Err: err}, nil
	}
	return teamPolicyResponse{Policy: resp}, nil
}

func (svc Service) NewTeamPolicy(ctx context.Context, teamID uint, queryID uint, resolution string) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{TeamID: ptr.Uint(teamID)}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.ds.NewTeamPolicy(ctx, teamID, queryID, resolution)
}

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listTeamPoliciesRequest struct {
	TeamID uint `url:"team_id"`
}

type listTeamPoliciesResponse struct {
	Policies []*fleet.Policy `json:"policies,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r listTeamPoliciesResponse) error() error { return r.Err }

func listTeamPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listTeamPoliciesRequest)
	resp, err := svc.ListTeamPolicies(ctx, req.TeamID)
	if err != nil {
		return listTeamPoliciesResponse{Err: err}, nil
	}
	return listTeamPoliciesResponse{Policies: resp}, nil
}

func (svc Service) ListTeamPolicies(ctx context.Context, teamID uint) ([]*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{TeamID: ptr.Uint(teamID)}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListTeamPolicies(ctx, teamID)
}

/////////////////////////////////////////////////////////////////////////////////
// Get by id
/////////////////////////////////////////////////////////////////////////////////

type getTeamPolicyByIDRequest struct {
	TeamID   uint `url:"team_id"`
	PolicyID uint `url:"policy_id"`
}

type getTeamPolicyByIDResponse struct {
	Policy *fleet.Policy `json:"policy"`
	Err    error         `json:"error,omitempty"`
}

func (r getTeamPolicyByIDResponse) error() error { return r.Err }

func getTeamPolicyByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getTeamPolicyByIDRequest)
	teamPolicy, err := svc.GetTeamPolicyByIDQueries(ctx, req.TeamID, req.PolicyID)
	if err != nil {
		return getTeamPolicyByIDResponse{Err: err}, nil
	}
	return getTeamPolicyByIDResponse{Policy: teamPolicy}, nil
}

func (svc Service) GetTeamPolicyByIDQueries(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{TeamID: ptr.Uint(teamID)}, fleet.ActionRead); err != nil {
		return nil, err
	}

	teamPolicy, err := svc.ds.TeamPolicy(ctx, teamID, policyID)
	if err != nil {
		return nil, err
	}

	return teamPolicy, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete
/////////////////////////////////////////////////////////////////////////////////

type deleteTeamPoliciesRequest struct {
	TeamID uint   `url:"team_id"`
	IDs    []uint `json:"ids"`
}

type deleteTeamPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r deleteTeamPoliciesResponse) error() error { return r.Err }

func deleteTeamPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteTeamPoliciesRequest)
	resp, err := svc.DeleteTeamPolicies(ctx, req.TeamID, req.IDs)
	if err != nil {
		return deleteTeamPoliciesResponse{Err: err}, nil
	}
	return deleteTeamPoliciesResponse{Deleted: resp}, nil
}

func (svc Service) DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{TeamID: ptr.Uint(teamID)}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.ds.DeleteTeamPolicies(ctx, teamID, ids)
}
