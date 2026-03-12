package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/baselines"
	"gopkg.in/yaml.v2"
)

const baselineNamePrefix = "[NVIDIA Baseline] "

// baselinePolicyYAML matches the YAML structure of verification policy files.
type baselinePolicyYAML struct {
	Name        string `yaml:"name"`
	Query       string `yaml:"query"`
	Description string `yaml:"description"`
	Resolution  string `yaml:"resolution"`
	Platform    string `yaml:"platform"`
	Critical    bool   `yaml:"critical"`
}

// List baselines

type listBaselinesRequest struct{}

type listBaselinesResponse struct {
	Baselines []fleet.BaselineManifest `json:"baselines"`
	Err       error                    `json:"error,omitempty"`
}

func (r listBaselinesResponse) Error() error { return r.Err }

func listBaselinesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
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

func applyBaselineEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyBaselineRequest)

	manifest, err := baselines.GetBaseline(req.BaselineID)
	if err != nil {
		return &applyBaselineResponse{Err: err}, nil
	}

	// Get team name for policy specs.
	team, err := svc.GetTeam(ctx, req.TeamID)
	if err != nil {
		return &applyBaselineResponse{Err: fmt.Errorf("looking up team %d: %w", req.TeamID, err)}, nil
	}

	// --- Remove existing baseline profiles for idempotent re-apply ---
	if err := removeBaselineProfiles(ctx, svc, req.TeamID); err != nil {
		return &applyBaselineResponse{Err: fmt.Errorf("removing old baseline profiles: %w", err)}, nil
	}

	// --- Remove existing baseline scripts for idempotent re-apply ---
	if err := removeBaselineScripts(ctx, svc, req.TeamID); err != nil {
		return &applyBaselineResponse{Err: fmt.Errorf("removing old baseline scripts: %w", err)}, nil
	}

	var profileNames, policyNames, scriptNames []string

	// --- Create profiles ---
	for _, cat := range manifest.Categories {
		for _, p := range cat.Profiles {
			content, err := baselines.GetProfileContent(req.BaselineID, p)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			profileName := baselineNamePrefix + cat.Name + " - " + stripExtension(p)
			_, err = svc.NewMDMWindowsConfigProfile(ctx, req.TeamID, profileName, content, nil, "")
			if err != nil {
				return &applyBaselineResponse{Err: fmt.Errorf("creating profile %s: %w", profileName, err)}, nil
			}
			profileNames = append(profileNames, profileName)
		}
	}

	// --- Create scripts and collect IDs for policy linking ---
	scriptIDs := make(map[string]uint) // script filename → script ID
	for _, cat := range manifest.Categories {
		for _, s := range cat.Scripts {
			content, err := baselines.GetScriptContent(req.BaselineID, s)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			scriptName := baselineNamePrefix + stripExtension(s)
			tmID := req.TeamID
			created, err := svc.NewScript(ctx, &tmID, scriptName, bytes.NewReader(content))
			if err != nil {
				return &applyBaselineResponse{Err: fmt.Errorf("creating script %s: %w", scriptName, err)}, nil
			}
			scriptIDs[s] = created.ID
			scriptNames = append(scriptNames, scriptName)
		}
	}

	// --- Create policies ---
	var allSpecs []*fleet.PolicySpec
	for _, cat := range manifest.Categories {
		for _, p := range cat.Policies {
			content, err := baselines.GetPolicyContent(req.BaselineID, p)
			if err != nil {
				return &applyBaselineResponse{Err: err}, nil
			}
			var policies []baselinePolicyYAML
			if err := yaml.Unmarshal(content, &policies); err != nil {
				return &applyBaselineResponse{Err: fmt.Errorf("parsing policy %s: %w", p, err)}, nil
			}

			// Find the remediation script for this category (if any).
			var scriptID *uint
			for _, s := range cat.Scripts {
				if id, ok := scriptIDs[s]; ok {
					scriptID = &id
					break
				}
			}

			for _, pol := range policies {
				spec := &fleet.PolicySpec{
					Name:        pol.Name,
					Query:       pol.Query,
					Description: pol.Description,
					Resolution:  pol.Resolution,
					Platform:    pol.Platform,
					Critical:    pol.Critical,
					Team:        team.Name,
					ScriptID:    scriptID,
				}
				allSpecs = append(allSpecs, spec)
				policyNames = append(policyNames, pol.Name)
			}
		}
	}

	if len(allSpecs) > 0 {
		if err := svc.ApplyPolicySpecs(ctx, allSpecs); err != nil {
			return &applyBaselineResponse{Err: fmt.Errorf("applying policies: %w", err)}, nil
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

func removeBaselineEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*removeBaselineRequest)

	// Remove baseline profiles.
	if err := removeBaselineProfiles(ctx, svc, req.TeamID); err != nil {
		return &removeBaselineResponse{Err: fmt.Errorf("removing baseline profiles: %w", err)}, nil
	}

	// Remove baseline scripts.
	if err := removeBaselineScripts(ctx, svc, req.TeamID); err != nil {
		return &removeBaselineResponse{Err: fmt.Errorf("removing baseline scripts: %w", err)}, nil
	}

	// Remove baseline policies.
	if err := removeBaselinePolicies(ctx, svc, req.TeamID); err != nil {
		return &removeBaselineResponse{Err: fmt.Errorf("removing baseline policies: %w", err)}, nil
	}

	return &removeBaselineResponse{}, nil
}

// removeBaselineProfiles deletes all profiles with the baseline name prefix for the given team.
func removeBaselineProfiles(ctx context.Context, svc fleet.Service, teamID uint) error {
	tmID := teamID
	profiles, _, err := svc.ListMDMConfigProfiles(ctx, &tmID, fleet.ListOptions{PerPage: 9999})
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if strings.HasPrefix(p.Name, baselineNamePrefix) {
			if err := svc.DeleteMDMWindowsConfigProfile(ctx, p.ProfileUUID); err != nil {
				return fmt.Errorf("deleting profile %s: %w", p.Name, err)
			}
		}
	}
	return nil
}

// removeBaselineScripts deletes all scripts with the baseline name prefix for the given team.
func removeBaselineScripts(ctx context.Context, svc fleet.Service, teamID uint) error {
	tmID := teamID
	scripts, _, err := svc.ListScripts(ctx, &tmID, fleet.ListOptions{PerPage: 9999})
	if err != nil {
		return err
	}
	for _, s := range scripts {
		if strings.HasPrefix(s.Name, baselineNamePrefix) {
			if err := svc.DeleteScript(ctx, s.ID); err != nil {
				return fmt.Errorf("deleting script %s: %w", s.Name, err)
			}
		}
	}
	return nil
}

// removeBaselinePolicies deletes all policies with the baseline name prefix for the given team.
func removeBaselinePolicies(ctx context.Context, svc fleet.Service, teamID uint) error {
	policies, _, err := svc.ListTeamPolicies(ctx, teamID, fleet.ListOptions{PerPage: 9999}, fleet.ListOptions{}, false)
	if err != nil {
		return err
	}
	var idsToDelete []uint
	for _, p := range policies {
		if strings.HasPrefix(p.Name, baselineNamePrefix) {
			idsToDelete = append(idsToDelete, p.ID)
		}
	}
	if len(idsToDelete) > 0 {
		if _, err := svc.DeleteTeamPolicies(ctx, teamID, idsToDelete); err != nil {
			return err
		}
	}
	return nil
}

// stripExtension removes the directory prefix and file extension from a path.
// e.g., "profiles/firewall.xml" → "firewall"
func stripExtension(p string) string {
	base := filepath.Base(p)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
