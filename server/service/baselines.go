package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/baselines"
)

// List baselines

type listBaselinesRequest struct{}

type listBaselinesResponse struct {
	Baselines []fleet.BaselineManifest `json:"baselines"`
	Err       error                    `json:"error,omitempty"`
}

func (r listBaselinesResponse) Error() error { return r.Err }

func listBaselinesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	all, err := baselines.ListBaselines()
	if err != nil {
		return &listBaselinesResponse{Err: err}, nil
	}
	return &listBaselinesResponse{Baselines: all}, nil
}

// Apply baseline

type applyBaselineRequest struct {
	BaselineID string `json:"baseline_id"`
	TeamID     uint   `json:"team_id"`
}

type applyBaselineResponse struct {
	BaselineID      string   `json:"baseline_id"`
	TeamID          uint     `json:"team_id"`
	ProfilesCreated []string `json:"profiles_created"`
	PoliciesCreated []string `json:"policies_created"`
	ScriptsCreated  []string `json:"scripts_created"`
	Err             error    `json:"error,omitempty"`
}

func (r applyBaselineResponse) Error() error { return r.Err }
func (r applyBaselineResponse) Status() int  { return http.StatusOK }

func applyBaselineEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyBaselineRequest)

	manifest, err := baselines.GetBaseline(req.BaselineID)
	if err != nil {
		return &applyBaselineResponse{Err: err}, nil
	}

	var profileNames, policyNames, scriptNames []string

	for _, cat := range manifest.Categories {
		for _, p := range cat.Profiles {
			content, err := baselines.GetProfileContent(req.BaselineID, p)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			_ = content // TODO(5.2): call svc.BatchSetMDMProfiles or individual profile upload
			profileNames = append(profileNames, cat.Name+" - "+p)
		}
		for _, p := range cat.Policies {
			content, err := baselines.GetPolicyContent(req.BaselineID, p)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			_ = content // TODO(5.2): call svc.ApplyPolicies
			policyNames = append(policyNames, cat.Name+" - "+p)
		}
		for _, s := range cat.Scripts {
			content, err := baselines.GetScriptContent(req.BaselineID, s)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			_ = content // TODO(5.2): call svc.BatchSetScripts
			scriptNames = append(scriptNames, cat.Name+" - "+s)
		}
	}

	return &applyBaselineResponse{
		BaselineID:      req.BaselineID,
		TeamID:          req.TeamID,
		ProfilesCreated: profileNames,
		PoliciesCreated: policyNames,
		ScriptsCreated:  scriptNames,
	}, nil
}

// Remove baseline

type removeBaselineRequest struct {
	BaselineID string `url:"baseline_id"`
	TeamID     uint   `url:"team_id"`
}

type removeBaselineResponse struct {
	Err error `json:"error,omitempty"`
}

func (r removeBaselineResponse) Error() error { return r.Err }

func removeBaselineEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	// TODO(5.3): implement removal of baseline-created resources
	// 1. Find profiles/policies/scripts with matching baseline name prefix for the team
	// 2. Delete them using existing service methods
	return &removeBaselineResponse{}, nil
}
