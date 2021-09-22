package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type globalPolicyRequest struct {
	QueryID uint `json:"query_id"`
}

type globalPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r globalPolicyResponse) error() error { return r.Err }

func globalPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*globalPolicyRequest)
	resp, err := svc.NewGlobalPolicy(ctx, req.QueryID)
	if err != nil {
		return globalPolicyResponse{Err: err}, nil
	}
	return globalPolicyResponse{Policy: resp}, nil
}

func (svc Service) NewGlobalPolicy(ctx context.Context, queryID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.ds.NewGlobalPolicy(ctx, queryID)
}

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listGlobalPoliciesResponse struct {
	Policies []*fleet.Policy `json:"policies,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r listGlobalPoliciesResponse) error() error { return r.Err }

func listGlobalPoliciesEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (interface{}, error) {
	resp, err := svc.ListGlobalPolicies(ctx)
	if err != nil {
		return listGlobalPoliciesResponse{Err: err}, nil
	}
	return listGlobalPoliciesResponse{Policies: resp}, nil
}

func (svc Service) ListGlobalPolicies(ctx context.Context) ([]*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListGlobalPolicies(ctx)
}

/////////////////////////////////////////////////////////////////////////////////
// Get by id
/////////////////////////////////////////////////////////////////////////////////

type getPolicyByIDRequest struct {
	PolicyID uint `url:"policy_id"`
}

type getPolicyByIDResponse struct {
	Policy *fleet.Policy `json:"policy"`
	Err    error         `json:"error,omitempty"`
}

func (r getPolicyByIDResponse) error() error { return r.Err }

func getPolicyByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getPolicyByIDRequest)
	policy, err := svc.GetPolicyByIDQueries(ctx, req.PolicyID)
	if err != nil {
		return getPolicyByIDResponse{Err: err}, nil
	}
	return getPolicyByIDResponse{Policy: policy}, nil
}

func (svc Service) GetPolicyByIDQueries(ctx context.Context, policyID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete
/////////////////////////////////////////////////////////////////////////////////

type deleteGlobalPoliciesRequest struct {
	IDs []uint `json:"ids"`
}

type deleteGlobalPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r deleteGlobalPoliciesResponse) error() error { return r.Err }

func deleteGlobalPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteGlobalPoliciesRequest)
	resp, err := svc.DeleteGlobalPolicies(ctx, req.IDs)
	if err != nil {
		return deleteGlobalPoliciesResponse{Err: err}, nil
	}
	return deleteGlobalPoliciesResponse{Deleted: resp}, nil
}

func (svc Service) DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.ds.DeleteGlobalPolicies(ctx, ids)
}
