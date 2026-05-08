package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// Get policy by id.
/////////////////////////////////////////////////////////////////////////////////

func getPolicyByIDEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetPolicyByIDRequest)
	policy, err := svc.GetPolicyByID(ctx, req.PolicyID)
	if err != nil {
		return fleet.GetPolicyByIDResponse{Err: err}, nil
	}
	return fleet.GetPolicyByIDResponse{Policy: policy}, nil
}

func (svc Service) GetPolicyByID(ctx context.Context, policyID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		return nil, err
	}

	// Re-authorize against the fetched policy so that team-scoped users cannot
	// read policies belonging to teams they have no role on. The initial
	// authorization above runs against an empty Policy{} (nil TeamID), which
	// permits any user with a team role to reach this point regardless of the
	// fetched policy's actual team.
	if err := svc.authz.Authorize(ctx, policy, fleet.ActionRead); err != nil {
		return nil, err
	}

	if err := svc.populateAutomationsForTeamPolicy(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate automations")
	}

	return policy, nil
}
