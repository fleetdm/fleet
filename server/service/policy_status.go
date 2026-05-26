package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Reset policy status
////////////////////////////////////////////////////////////////////////////////

func resetPolicyStatusEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ResetPolicyStatusRequest)
	if err := svc.ResetPolicyStatus(ctx, req.PolicyID); err != nil {
		return fleet.ResetPolicyStatusResponse{Err: err}, nil
	}
	return fleet.ResetPolicyStatusResponse{}, nil
}

func (svc Service) ResetPolicyStatus(ctx context.Context, policyID uint) error {
	// Load policy first to get TeamID for auth, same pattern as GetPolicyByID.
	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		svc.SkipAuth(ctx)
		return err
	}
	if err := svc.authz.Authorize(ctx, policy, fleet.ActionWrite); err != nil {
		return err
	}
	if !license.IsPremium(ctx) {
		return fleet.ErrMissingLicense
	}
	return svc.ds.ClearPolicyRuns(ctx, policyID)
}
