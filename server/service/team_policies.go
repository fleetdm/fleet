package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type teamPolicyRequest struct {
	TeamID      uint   `url:"team_id"`
	QueryID     *uint  `json:"query_id"`
	Query       string `json:"query"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
	Platform    string `json:"platform"`
}

type teamPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r teamPolicyResponse) error() error { return r.Err }

func teamPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*teamPolicyRequest)
	resp, err := svc.NewTeamPolicy(ctx, req.TeamID, fleet.PolicyPayload{
		QueryID:     req.QueryID,
		Name:        req.Name,
		Query:       req.Query,
		Description: req.Description,
		Resolution:  req.Resolution,
		Platform:    req.Platform,
	})
	if err != nil {
		return teamPolicyResponse{Err: err}, nil
	}
	return teamPolicyResponse{Policy: resp}, nil
}

func (svc Service) NewTeamPolicy(ctx context.Context, teamID uint, p fleet.PolicyPayload) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("user must be authenticated to create team policies")
	}

	if err := p.Verify(); err != nil {
		return nil, &badRequestError{
			message: fmt.Sprintf("policy payload verification: %s", err),
		}
	}
	policy, err := svc.ds.NewTeamPolicy(ctx, teamID, ptr.Uint(vc.UserID()), p)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating policy")
	}
	return policy, nil
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
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionRead); err != nil {
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
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionRead); err != nil {
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
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	ids, err := svc.ds.DeleteTeamPolicies(ctx, teamID, ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Modify
/////////////////////////////////////////////////////////////////////////////////

type modifyTeamPolicyRequest struct {
	TeamID   uint `url:"team_id"`
	PolicyID uint `url:"policy_id"`
	fleet.ModifyPolicyPayload
}

type modifyTeamPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r modifyTeamPolicyResponse) error() error { return r.Err }

func modifyTeamPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamPolicyRequest)
	resp, err := svc.ModifyTeamPolicy(ctx, req.TeamID, req.PolicyID, req.ModifyPolicyPayload)
	if err != nil {
		return modifyTeamPolicyResponse{Err: err}, nil
	}
	return modifyTeamPolicyResponse{Policy: resp}, nil
}

func (svc Service) ModifyTeamPolicy(ctx context.Context, teamID uint, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	return svc.modifyPolicy(ctx, &teamID, id, p)
}

func (svc Service) modifyPolicy(ctx context.Context, teamID *uint, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	// First make sure the user can read the policies.
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: teamID,
		},
	}, fleet.ActionRead); err != nil {
		return nil, err
	}
	policy, err := svc.ds.Policy(ctx, id)
	if err != nil {
		return nil, err
	}
	// Then we make sure they can modify the team's policies.
	if err := svc.authz.Authorize(ctx, policy, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := p.Verify(); err != nil {
		return nil, &badRequestError{
			message: fmt.Sprintf("policy payload verification: %s", err),
		}
	}

	if p.Name != nil {
		policy.Name = *p.Name
	}
	if p.Description != nil {
		policy.Description = *p.Description
	}
	if p.Query != nil {
		policy.Query = *p.Query
	}
	if p.Resolution != nil {
		policy.Resolution = p.Resolution
	}
	logging.WithExtras(ctx, "name", policy.Name, "sql", policy.Query)

	err = svc.ds.SavePolicy(ctx, policy)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving policy")
	}

	return policy, nil
}
