package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ///////////////////////////////////////////////////////////////////////////////
// Get policy status
// ///////////////////////////////////////////////////////////////////////////////

func getPolicyStatusEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	if !license.IsPremium(ctx) {
		return nil, fleet.ErrMissingLicense
	}

	req := request.(*fleet.GetPolicyStatusRequest)
	policy, err := svc.GetPolicyByID(ctx, req.PolicyID)
	if err != nil {
		return &fleet.GetPolicyStatusResponse{Err: err}, nil
	}

	return svc.GetPolicyStatus(ctx, policy, *req)
}

func (svc Service) GetPolicyStatus(ctx context.Context, policy *fleet.Policy, req fleet.GetPolicyStatusRequest) (*fleet.GetPolicyStatusResponse, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	switch req.RunStatus {
	case "", "policy_failed", "automation_failed":
		// valid
	default:
		return nil, fleet.NewInvalidArgumentError("run_status", `must be one of "policy_failed", "automation_failed"`)
	}

	// Default to newest-first so pages are stable when the caller omits order_key.
	if req.ListOptions.OrderKey == "" {
		req.ListOptions.OrderKey = "created_at"
		req.ListOptions.OrderDirection = fleet.OrderDescending
	}

	// IncludeObserver:true so team observers can read policy status for teams
	// they observe; the calling user has already been authorized to read the
	// policy itself via GetPolicyByID.
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	runs, count, meta, err := svc.ds.GetPolicyStatus(ctx, policy.ID, filter, req)
	if err != nil {
		return nil, err
	}

	return &fleet.GetPolicyStatusResponse{
		Runs:  runs,
		Count: count,
		Meta:  meta,
	}, nil
}
