package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type globalPolicyRequest struct {
	QueryID     *uint  `json:"query_id"`
	Query       string `json:"query"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
	Platform    string `json:"platform"`
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
		Platform:    req.Platform,
	})
	if err != nil {
		return globalPolicyResponse{Err: err}, nil
	}
	return globalPolicyResponse{Policy: resp}, nil
}

func (svc Service) NewGlobalPolicy(ctx context.Context, p fleet.PolicyPayload) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("user must be authenticated to create team policies")
	}
	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &badRequestError{
			message: fmt.Sprintf("policy payload verification: %s", err),
		})
	}
	policy, err := svc.ds.NewGlobalPolicy(ctx, ptr.Uint(vc.UserID()), p)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "storing policy")
	}
	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedPolicy,
		&map[string]interface{}{"policy_id": policy.ID, "policy_name": policy.Name},
	); err != nil {
		return nil, err
	}
	return policy, nil
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

// DeleteGlobalPolicies deletes the given policies from the database.
// It also deletes the given ids from the failing policies webhook configuration.
func (svc Service) DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error) {
	// First check if authorized to read policies
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	policiesByID, err := svc.ds.PoliciesByID(ctx, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting policies by ID")
	}
	// Then check if authorized to write policies
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	for _, policy := range policiesByID {
		if policy.PolicyData.TeamID != nil {
			return nil, authz.ForbiddenWithInternal(
				"attempting to delete policy that belongs to team",
				authz.UserFromContext(ctx),
				policy,
				fleet.ActionWrite,
			)
		}
	}
	if err := svc.removeGlobalPoliciesFromWebhookConfig(ctx, ids); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "removing global policies from webhook config")
	}
	deletedIDs, err := svc.ds.DeleteGlobalPolicies(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	for _, id := range deletedIDs {
		if err := svc.ds.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeDeletedPolicy,
			&map[string]interface{}{"policy_id": id, "policy_name": policiesByID[id].Name},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding new activity for deleted policy")
		}
	}
	return ids, nil
}

func (svc Service) removeGlobalPoliciesFromWebhookConfig(ctx context.Context, ids []uint) error {
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	idSet := make(map[uint]struct{})
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	n := 0
	policyIDs := ac.WebhookSettings.FailingPoliciesWebhook.PolicyIDs
	origLen := len(policyIDs)
	for i := range policyIDs {
		if _, ok := idSet[policyIDs[i]]; !ok {
			policyIDs[n] = policyIDs[i]
			n++
		}
	}
	if n == origLen {
		return nil
	}
	ac.WebhookSettings.FailingPoliciesWebhook.PolicyIDs = policyIDs[:n]
	if err := svc.ds.SaveAppConfig(ctx, ac); err != nil {
		return err
	}
	return nil
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

func (svc *Service) ModifyGlobalPolicy(ctx context.Context, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
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

// TODO: add tests for activities?
func (svc *Service) ApplyPolicySpecs(ctx context.Context, policies []*fleet.PolicySpec) error {
	checkGlobalPolicyAuth := false
	for _, policy := range policies {
		if err := policy.Verify(); err != nil {
			return ctxerr.Wrap(ctx, &badRequestError{
				message: fmt.Sprintf("policy spec payload verification: %s", err),
			})
		}
		if policy.Team != "" {
			team, err := svc.ds.TeamByName(ctx, policy.Team)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting team by name")
			}
			if err := svc.authz.Authorize(ctx, &fleet.Policy{
				PolicyData: fleet.PolicyData{
					TeamID: &team.ID,
				},
			}, fleet.ActionWrite); err != nil {
				return err
			}
		} else {
			checkGlobalPolicyAuth = true
		}
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
	if err := svc.ds.ApplyPolicySpecs(ctx, vc.UserID(), policies); err != nil {
		return ctxerr.Wrap(ctx, err, "applying policy specs")
	}
	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAppliedSpecPolicy,
		&map[string]interface{}{"policies": policies},
	)
}
