package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// maxPolicyAutomationActivitiesPerPage is the upper bound for per_page on the
// list-policy-automation-activities endpoint.
const maxPolicyAutomationActivitiesPerPage = 10_000

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
	// First, fetch policy to extract team for authorization checks.
	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		svc.SkipAuth(ctx)
		return nil, err
	}

	// Now, authorize against the fetched policy so that team-scoped users cannot
	// read policies belonging to teams they have no role on.
	if err := svc.authz.Authorize(ctx, policy, fleet.ActionRead); err != nil {
		return nil, err
	}

	// If it's a team policy we populate the automations on the policy.
	if err := svc.populateAutomationsForTeamPolicy(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate automations")
	}
	if err := svc.populateSoftwareIconURLs(ctx, []*fleet.Policy{policy}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate software icon urls")
	}

	return policy, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Reset policy.
/////////////////////////////////////////////////////////////////////////////////

func resetPolicyEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ResetPolicyRequest)
	err := svc.ResetPolicy(ctx, req.PolicyID)
	return fleet.ResetPolicyResponse{Err: err}, nil
}

func (svc Service) ResetPolicy(ctx context.Context, policyID uint) error {
	// Load first to authorize against the policy's actual team.
	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		svc.SkipAuth(ctx)
		return err
	}
	if err := svc.authz.Authorize(ctx, policy, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.ResetPolicy(ctx, policyID); err != nil {
		return ctxerr.Wrap(ctx, err, "reset policy")
	}

	var activityTeamID *int64
	var teamName *string
	switch {
	case policy.TeamID == nil:
		id := int64(-1)
		activityTeamID = &id
	case *policy.TeamID == 0:
		id := int64(0)
		activityTeamID = &id
	default:
		id := int64(*policy.TeamID) //nolint:gosec // policy team IDs are small
		activityTeamID = &id
		if svc.EnterpriseOverrides != nil && svc.EnterpriseOverrides.TeamByIDOrName != nil {
			team, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, policy.TeamID, nil)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "fetching team details")
			}
			teamName = &team.Name
		}
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeResetPolicy{
		ID:       policy.ID,
		Name:     policy.Name,
		TeamID:   activityTeamID,
		TeamName: teamName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for policy reset")
	}
	return nil
}

/////////////////////////////////////////////////////////////////////////////////
// List policy automation activities.
/////////////////////////////////////////////////////////////////////////////////

func listPolicyAutomationActivitiesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListPolicyAutomationActivitiesRequest)
	activities, meta, err := svc.ListPolicyAutomationActivities(ctx, req.PolicyID, req.Opts, req.Status)
	if err != nil {
		return fleet.ListPolicyAutomationActivitiesResponse{Err: err}, nil
	}
	resp := fleet.ListPolicyAutomationActivitiesResponse{
		Activities: activities,
		Meta:       meta,
	}
	if meta != nil {
		resp.Count = meta.TotalResults
	}
	return resp, nil
}

func (svc Service) ListPolicyAutomationActivities(ctx context.Context, policyID uint, opts fleet.ListOptions, status string) ([]*fleet.PolicyAutomationActivity, *fleet.PaginationMetadata, error) {
	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		svc.SkipAuth(ctx)
		return nil, nil, err
	}

	if err := svc.authz.Authorize(ctx, policy, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	switch status {
	case "", "error", "success":
		// valid
	default:
		return nil, nil, fleet.NewInvalidArgumentError("status", `must be "error", "success", or empty`)
	}

	if opts.PerPage == 0 {
		opts.PerPage = 50
	} else if opts.PerPage > maxPolicyAutomationActivitiesPerPage {
		return nil, nil, fleet.NewInvalidArgumentError("per_page", fmt.Sprintf("must be no greater than %d", maxPolicyAutomationActivitiesPerPage))
	}
	if opts.OrderKey == "" {
		// Default to newest activity first.
		opts.OrderKey = "created_at"
		opts.OrderDirection = fleet.OrderDescending
	}
	opts.IncludeMetadata = true

	activities, meta, err := svc.ds.ListPolicyAutomationActivities(ctx, policyID, filter, opts, status)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list policy automation activities")
	}
	return activities, meta, nil
}
