package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type globalPolicyRequest struct {
	QueryID     uint   `json:"query_id"`
	Query       string `json:"query"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
}

type globalPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r globalPolicyResponse) error() error { return r.Err }

func globalPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*globalPolicyRequest)
	resp, err := svc.NewGlobalPolicy(ctx, fleet.PolicyPayload{
		QueryID:     req.QueryID,
		Query:       req.Query,
		Name:        req.Name,
		Description: req.Description,
		Resolution:  req.Resolution,
	})
	if err != nil {
		return globalPolicyResponse{Err: err}, nil
	}
	return globalPolicyResponse{Policy: resp}, nil
}

type NewPolicyArgs struct {
	QueryID     uint
	Query       string
	Name        string
	Description string
	Resolution  string
}

func (svc Service) NewGlobalPolicy(ctx context.Context, p fleet.PolicyPayload) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("user must be authenticated to create team policies")
	}
	if p.QueryID != 0 && p.Query != "" {
		return nil, ctxerr.New(ctx, "both fields \"queryID\" and \"query\" cannot be set")
	}
	// TODO(lucas): Implement me.
	// if err := p.ValidateSQL(); err != nil {
	//	return nil, err
	// }

	// TODO(lucas): Add activity entry.

	return svc.ds.NewGlobalPolicy(ctx, vc.UserID(), p.QueryID, p.Name, p.Query, p.Description, p.Resolution)
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

/////////////////////////////////////////////////////////////////////////////////
// Modify
/////////////////////////////////////////////////////////////////////////////////

type modifyGlobalPolicyRequest struct {
	PolicyID uint `url:"policy_id"`
	fleet.ModifyPolicyPayload
}

type modifyGlobalPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r modifyGlobalPolicyResponse) error() error { return r.Err }

func modifyGlobalPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyGlobalPolicyRequest)
	resp, err := svc.ModifyGlobalPolicy(ctx, req.PolicyID, req.ModifyPolicyPayload)
	if err != nil {
		return modifyGlobalPolicyResponse{Err: err}, nil
	}
	return modifyGlobalPolicyResponse{Policy: resp}, nil
}

func (svc Service) ModifyGlobalPolicy(ctx context.Context, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	return svc.modifyPolicy(ctx, nil, id, p)
}

/////////////////////////////////////////////////////////////////////////////////
// Apply Spec
/////////////////////////////////////////////////////////////////////////////////

type applyPolicySpecsRequest struct {
	Specs []*fleet.PolicySpec `json:"specs"`
}

type applyPolicySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyPolicySpecsResponse) error() error { return r.Err }

func applyPolicySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyPolicySpecsRequest)
	err := svc.ApplyPolicySpecs(ctx, req.Specs)
	if err != nil {
		return applyPolicySpecsResponse{Err: err}, nil
	}
	return applyPolicySpecsResponse{}, nil
}

func (svc Service) ApplyPolicySpecs(ctx context.Context, policies []*fleet.PolicySpec) error {
	checkGlobalPolicyAuth := false
	for _, policy := range policies {
		if policy.Team != "" {
			team, err := svc.ds.TeamByName(ctx, policy.Team)
			if err != nil {
				return errors.Wrap(err, "getting team by name")
			}
			if err := svc.authz.Authorize(ctx, &fleet.Policy{TeamID: &team.ID}, fleet.ActionWrite); err != nil {
				return err
			}
			continue
		}
		checkGlobalPolicyAuth = true
	}
	if checkGlobalPolicyAuth {
		if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
			return err
		}
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errors.New("user must be authenticated to apply policies")
	}

	return svc.ds.ApplyPolicySpecs(ctx, vc.UserID(), policies)
}
