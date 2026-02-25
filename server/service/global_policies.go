package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// Add
func globalPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GlobalPolicyRequest)
	resp, err := svc.NewGlobalPolicy(ctx, fleet.PolicyPayload{
		QueryID:          req.QueryID,
		Query:            req.Query,
		Name:             req.Name,
		Description:      req.Description,
		Resolution:       req.Resolution,
		Platform:         req.Platform,
		Critical:         req.Critical,
		LabelsIncludeAny: req.LabelsIncludeAny,
		LabelsExcludeAny: req.LabelsExcludeAny,
	})
	if err != nil {
		return fleet.GlobalPolicyResponse{Err: err}, nil
	}
	return fleet.GlobalPolicyResponse{Policy: resp}, nil
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
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("policy payload verification: %s", err),
		})
	}

	if err := verifyLabelsToAssociate(ctx, svc.ds, nil, append(p.LabelsIncludeAny, p.LabelsExcludeAny...), vc.User); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "verify labels to associate")
	}

	policy, err := svc.ds.NewGlobalPolicy(ctx, ptr.Uint(vc.UserID()), p)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "storing policy")
	}
	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	globalTeamID := int64(-1)
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedPolicy{
			ID:       policy.ID,
			Name:     policy.Name,
			TeamID:   &globalTeamID,
			TeamName: nil,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for global policy creation")
	}
	return policy, nil
}

// List
func listGlobalPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListGlobalPoliciesRequest)
	resp, err := svc.ListGlobalPolicies(ctx, req.Opts)
	if err != nil {
		return fleet.ListGlobalPoliciesResponse{Err: err}, nil
	}
	return fleet.ListGlobalPoliciesResponse{Policies: resp}, nil
}

func (svc Service) ListGlobalPolicies(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListGlobalPolicies(ctx, opts)
}

// Get by id
func getPolicyByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetPolicyByIDRequest)
	policy, err := svc.GetPolicyByIDQueries(ctx, req.PolicyID)
	if err != nil {
		return fleet.GetPolicyByIDResponse{Err: err}, nil
	}
	return fleet.GetPolicyByIDResponse{Policy: policy}, nil
}

func (svc Service) GetPolicyByIDQueries(ctx context.Context, policyID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	policy, err := svc.ds.Policy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	if err := svc.populatePolicyInstallSoftware(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate install_software")
	}
	if err := svc.populatePolicyRunScript(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate run_script")
	}

	return policy, nil
}

// ///////////////////////////////////////////////////////////////////////////////
// Count
// ///////////////////////////////////////////////////////////////////////////////

func countGlobalPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CountGlobalPoliciesRequest)
	resp, err := svc.CountGlobalPolicies(ctx, req.ListOptions.MatchQuery)
	if err != nil {
		return fleet.CountGlobalPoliciesResponse{Err: err}, nil
	}
	return fleet.CountGlobalPoliciesResponse{Count: resp}, nil
}

func (svc Service) CountGlobalPolicies(ctx context.Context, matchQuery string) (int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionRead); err != nil {
		return 0, err
	}

	count, err := svc.ds.CountPolicies(ctx, nil, matchQuery)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Delete
func deleteGlobalPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteGlobalPoliciesRequest)
	resp, err := svc.DeleteGlobalPolicies(ctx, req.IDs)
	if err != nil {
		return fleet.DeleteGlobalPoliciesResponse{Err: err}, nil
	}
	return fleet.DeleteGlobalPoliciesResponse{Deleted: resp}, nil
}

// DeleteGlobalPolicies deletes the given policies from the database.
// It also deletes the given ids from the failing policies webhook configuration.
func (svc Service) DeleteGlobalPolicies(ctx context.Context, ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	policiesByID, err := svc.ds.PoliciesByID(ctx, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting policies by ID")
	}
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
		globalTeamID := int64(-1)
		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeDeletedPolicy{
				ID:       id,
				Name:     policiesByID[id].Name,
				TeamID:   &globalTeamID,
				TeamName: nil,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for policy deletion")
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

// Modify
func modifyGlobalPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyGlobalPolicyRequest)
	resp, err := svc.ModifyGlobalPolicy(ctx, req.PolicyID, req.ModifyPolicyPayload)
	if err != nil {
		return fleet.ModifyGlobalPolicyResponse{Err: err}, nil
	}
	return fleet.ModifyGlobalPolicyResponse{Policy: resp}, nil
}

func (svc *Service) ModifyGlobalPolicy(ctx context.Context, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	return svc.modifyPolicy(ctx, nil, id, p)
}

// Reset automation
func resetAutomationEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ResetAutomationRequest)
	err := svc.ResetAutomation(ctx, req.TeamIDs, req.PolicyIDs)
	return fleet.ResetAutomationResponse{Err: err}, nil
}

func (svc *Service) ResetAutomation(ctx context.Context, teamIDs, policyIDs []uint) error {
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	allAutoPolicies := automationPolicies(ac.WebhookSettings.FailingPoliciesWebhook, ac.Integrations.Jira, ac.Integrations.Zendesk)
	pIDs := make(map[uint]struct{})
	for _, id := range policyIDs {
		pIDs[id] = struct{}{}
	}
	for _, teamID := range teamIDs {
		p1, p2, err := svc.ds.ListTeamPolicies(ctx, teamID, fleet.ListOptions{}, fleet.ListOptions{})
		if err != nil {
			return err
		}
		for _, p := range p1 {
			pIDs[p.ID] = struct{}{}
		}
		for _, p := range p2 {
			pIDs[p.ID] = struct{}{}
		}
	}
	hasGlobal := false
	tIDs := make(map[uint]struct{})
	for id := range pIDs {
		p, err := svc.ds.Policy(ctx, id)
		if err != nil {
			return err
		}
		if p.TeamID == nil {
			hasGlobal = true
		} else {
			tIDs[*p.TeamID] = struct{}{}
		}
	}
	for id := range tIDs {
		var teamConfig fleet.TeamConfigLite
		if id == 0 {
			// Handle "No Team" (team ID 0) - use AppConfig authorization
			if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
				return err
			}
			defaultConfig, err := svc.ds.DefaultTeamConfig(ctx)
			if err != nil {
				return err
			}
			teamConfig = defaultConfig.ToLite()
		} else {
			// Handle regular teams
			if err := svc.authz.Authorize(ctx, &fleet.Team{ID: id}, fleet.ActionWrite); err != nil {
				return err
			}
			t, err := svc.ds.TeamLite(ctx, id)
			if err != nil {
				return err
			}
			teamConfig = t.Config
		}
		for pID := range teamAutomationPolicies(teamConfig.WebhookSettings.FailingPoliciesWebhook, teamConfig.Integrations.Jira, teamConfig.Integrations.Zendesk) {
			allAutoPolicies[pID] = struct{}{}
		}
	}
	if hasGlobal {
		if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
			return err
		}
	}
	if len(tIDs) == 0 && !hasGlobal {
		svc.authz.SkipAuthorization(ctx)
		return nil
	}
	for id := range pIDs {
		if _, ok := allAutoPolicies[id]; !ok {
			continue
		}
		if err := svc.ds.IncreasePolicyAutomationIteration(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func automationPolicies(wh fleet.FailingPoliciesWebhookSettings, ji []*fleet.JiraIntegration, zi []*fleet.ZendeskIntegration) map[uint]struct{} {
	enabled := wh.Enable
	for _, j := range ji {
		if j.EnableFailingPolicies {
			enabled = true
		}
	}
	for _, z := range zi {
		if z.EnableFailingPolicies {
			enabled = true
		}
	}
	pols := make(map[uint]struct{}, len(wh.PolicyIDs))
	if !enabled {
		return pols
	}
	for _, pid := range wh.PolicyIDs {
		pols[pid] = struct{}{}
	}
	return pols
}

func teamAutomationPolicies(wh fleet.FailingPoliciesWebhookSettings, ji []*fleet.TeamJiraIntegration, zi []*fleet.TeamZendeskIntegration) map[uint]struct{} {
	enabled := wh.Enable
	for _, j := range ji {
		if j.EnableFailingPolicies {
			enabled = true
		}
	}
	for _, z := range zi {
		if z.EnableFailingPolicies {
			enabled = true
		}
	}
	pols := make(map[uint]struct{}, len(wh.PolicyIDs))
	if !enabled {
		return pols
	}
	for _, pid := range wh.PolicyIDs {
		pols[pid] = struct{}{}
	}
	return pols
}

// Apply Spec
func applyPolicySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ApplyPolicySpecsRequest)
	err := svc.ApplyPolicySpecs(ctx, req.Specs)
	if err != nil {
		return fleet.ApplyPolicySpecsResponse{Err: err}, nil
	}
	return fleet.ApplyPolicySpecsResponse{}, nil
}

// checkPolicySpecAuthorization verifies that the user is authorized to modify the
// policies defined in the spec, and returns a map from team names to team IDs if successful
func (svc *Service) checkPolicySpecAuthorization(ctx context.Context, policies []*fleet.PolicySpec) (map[string]uint, error) {
	checkGlobalPolicyAuth := false
	teamIDsByName := make(map[string]uint)
	for _, policy := range policies {
		if policy.Team != "" && policy.Team != "No team" {
			team, err := svc.ds.TeamByName(ctx, policy.Team)
			if err != nil {
				// This is so that the proper HTTP status code is returned
				svc.authz.SkipAuthorization(ctx)
				return nil, ctxerr.Wrap(ctx, err, "getting team by name")
			}
			if err := svc.authz.Authorize(ctx, &fleet.Policy{
				PolicyData: fleet.PolicyData{
					TeamID: &team.ID,
				},
			}, fleet.ActionWrite); err != nil {
				return nil, err
			}

			teamIDsByName[policy.Team] = team.ID
		} else {
			checkGlobalPolicyAuth = true
		}
	}
	if checkGlobalPolicyAuth {
		if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
			return nil, err
		}
	}
	return teamIDsByName, nil
}

func (svc *Service) ApplyPolicySpecs(ctx context.Context, policies []*fleet.PolicySpec) error {
	// Check authorization first.
	teamIDsByName, err := svc.checkPolicySpecAuthorization(ctx, policies)
	if err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errors.New("user must be authenticated to apply policies")
	}

	// After the authorization check, check the policy fields.
	for _, policy := range policies {
		if err := policy.Verify(); err != nil {
			return ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("policy spec payload verification: %s", err),
			})
		}

		// Make sure any applied labels exist.
		labels := policy.LabelsIncludeAny
		labels = append(labels, policy.LabelsExcludeAny...)
		if len(labels) > 0 {
			var teamID *uint       // ensure labels specified exist and are global or on the same team as the policy
			if policy.Team != "" { // if we get 0 as team ID, we'll pull only global labels, which is fine
				teamID = ptr.Uint(teamIDsByName[policy.Team])
			}

			labelsMap, err := svc.ds.LabelsByName(ctx, labels, fleet.TeamFilter{User: vc.User, TeamID: teamID})
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting labels by name")
			}
			for _, label := range labels {
				if _, ok := labelsMap[label]; !ok {
					return ctxerr.Wrap(ctx, &fleet.BadRequestError{
						Message: fmt.Sprintf("label %q does not exist, or cannot be applied to this policy", label),
					})
				}
			}
		}

	}

	// An empty string indicates there are no duplicate names.
	if name := fleet.FirstDuplicatePolicySpecName(policies); name != "" {
		return ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: "duplicate policy names not allowed",
		})
	}

	if !license.IsPremium(ctx) {
		for i := range policies {
			policies[i].Critical = false
		}
	}

	if err := svc.ds.ApplyPolicySpecs(ctx, vc.UserID(), policies); err != nil {
		return ctxerr.Wrap(ctx, err, "applying policy specs")
	}
	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAppliedSpecPolicy{
			Policies: policies,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for policy spec")
	}
	return nil
}

// Autofill
func autofillPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.AutofillPoliciesRequest)
	description, resolution, err := svc.AutofillPolicySql(ctx, req.SQL)
	return fleet.AutofillPoliciesResponse{Description: description, Resolution: resolution, Err: err}, nil
}

// Exposing external URL and timeout for testing purposes
var (
	getHumanInterpretationFromOsquerySqlUrl     = "https://fleetdm.com/api/v1/get-human-interpretation-from-osquery-sql"
	getHumanInterpretationFromOsquerySqlTimeout = 30 * time.Second
)

type AutofillError struct {
	Message     string
	InternalErr error
}

// Error implements the error interface.
func (e AutofillError) Error() string {
	return e.Message
}

// StatusCode implements the kithttp.StatusCoder interface.
func (e AutofillError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

func (e AutofillError) Internal() string {
	if e.InternalErr == nil {
		return ""
	}
	return e.InternalErr.Error()
}

func (svc *Service) AutofillPolicySql(ctx context.Context, sql string) (description string, resolution string, err error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		svc.authz.SkipAuthorization(ctx)
		return "", "", fleet.ErrNoContext
	}

	// We expect that only users with policy write permissions will autofill policies.
	if vc.User.GlobalRole != nil || len(vc.User.Teams) == 0 {
		if err = svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
			return "", "", err
		}
	} else {
		// Check if this user has team policy write permissions.
		teamID := vc.User.Teams[0].Team.ID
		for _, teamUser := range vc.User.Teams {
			if teamUser.Role == fleet.RoleAdmin || teamUser.Role == fleet.RoleMaintainer || teamUser.Role == fleet.RoleGitOps {
				teamID = teamUser.Team.ID
				break
			}
		}
		err = svc.authz.Authorize(
			ctx, &fleet.Policy{PolicyData: fleet.PolicyData{TeamID: &teamID}}, fleet.ActionWrite,
		)
		if err != nil {
			return "", "", err
		}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", "", err
	}
	if appConfig.ServerSettings.AIFeaturesDisabled {
		return "", "", ctxerr.Wrap(
			ctx, &fleet.BadRequestError{
				Message: "AI features are disabled (server_settings.ai_features_disabled)",
			},
		)
	}

	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "", "", ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "'sql' cannot be empty"})
	}

	// Using a timeout smaller than the Fleet server's WriteTimeout
	client := fleethttp.NewClient(fleethttp.WithTimeout(getHumanInterpretationFromOsquerySqlTimeout))
	reqBodyValues := map[string]string{"sql": sql}
	reqBody, err := json.Marshal(reqBodyValues)
	if err != nil {
		return "", "", ctxerr.Wrap(
			ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("Could not process sql: %s", sql),
			},
		)
	}
	resp, err := client.Post(
		getHumanInterpretationFromOsquerySqlUrl, "application/json", bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", "", ctxerr.Wrap(
			ctx, AutofillError{
				Message:     "error sending request to get human interpretation from osquery sql",
				InternalErr: err,
			},
		)
	}
	defer resp.Body.Close()
	if (resp.StatusCode / 100) != 2 {
		return "", "", ctxerr.Wrap(
			ctx, AutofillError{
				Message: "error from human interpretation of osquery sql",
				InternalErr: fmt.Errorf(
					"%s returned %d status code", getHumanInterpretationFromOsquerySqlUrl, resp.StatusCode,
				),
			},
		)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", ctxerr.Wrap(
			ctx, AutofillError{
				Message:     "error reading response body from human interpretation of osquery sql",
				InternalErr: err,
			},
		)
	}

	var result map[string]string
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", "", ctxerr.Wrap(
			ctx, AutofillError{
				Message:     "error unmarshaling response body from human interpretation of osquery sql",
				InternalErr: err,
			},
		)
	}
	const maxLength = 1<<16 - 1
	descriptionTrimmed := result["risks"]
	if len(descriptionTrimmed) > maxLength {
		descriptionTrimmed = descriptionTrimmed[:maxLength]
	}
	resolutionTrimmed := result["whatWillProbablyHappenDuringMaintenance"]
	if len(resolutionTrimmed) > maxLength {
		resolutionTrimmed = resolutionTrimmed[:maxLength]
	}
	return descriptionTrimmed, resolutionTrimmed, nil
}
