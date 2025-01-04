package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// Add
/////////////////////////////////////////////////////////////////////////////////

type teamPolicyRequest struct {
	TeamID                uint   `url:"team_id"`
	QueryID               *uint  `json:"query_id"`
	Query                 string `json:"query"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Resolution            string `json:"resolution"`
	Platform              string `json:"platform"`
	Critical              bool   `json:"critical" premium:"true"`
	CalendarEventsEnabled bool   `json:"calendar_events_enabled"`
	SoftwareTitleID       *uint  `json:"software_title_id"`
	ScriptID              *uint  `json:"script_id"`
}

type teamPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r teamPolicyResponse) error() error { return r.Err }

func teamPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*teamPolicyRequest)
	resp, err := svc.NewTeamPolicy(ctx, req.TeamID, fleet.NewTeamPolicyPayload{
		QueryID:               req.QueryID,
		Name:                  req.Name,
		Query:                 req.Query,
		Description:           req.Description,
		Resolution:            req.Resolution,
		Platform:              req.Platform,
		Critical:              req.Critical,
		CalendarEventsEnabled: req.CalendarEventsEnabled,
		SoftwareTitleID:       req.SoftwareTitleID,
		ScriptID:              req.ScriptID,
	})
	if err != nil {
		return teamPolicyResponse{Err: err}, nil
	}
	return teamPolicyResponse{Policy: resp}, nil
}

func (svc Service) NewTeamPolicy(ctx context.Context, teamID uint, tp fleet.NewTeamPolicyPayload) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("user must be authenticated to create team policies")
	}

	p, err := svc.newTeamPolicyPayloadToPolicyPayload(ctx, teamID, tp)
	if err != nil {
		return nil, err
	}

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("policy payload verification: %s", err),
		})
	}
	policy, err := svc.ds.NewTeamPolicy(ctx, teamID, ptr.Uint(vc.UserID()), p)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating policy")
	}

	if err := svc.populatePolicyInstallSoftware(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate install_software")
	}
	if err := svc.populatePolicyRunScript(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate run_script")
	}

	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedPolicy{
			ID:   policy.ID,
			Name: policy.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for team policy creation")
	}
	return policy, nil
}

func (svc *Service) populatePolicyInstallSoftware(ctx context.Context, p *fleet.Policy) error {
	if p.SoftwareInstallerID != nil {
		installerMetadata, err := svc.ds.GetSoftwareInstallerMetadataByID(ctx, *p.SoftwareInstallerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get software installer metadata by id")
		}
		p.InstallSoftware = &fleet.PolicySoftwareTitle{
			SoftwareTitleID: *installerMetadata.TitleID,
			Name:            installerMetadata.SoftwareTitle,
		}
		return nil
	}

	if p.VPPAdamID != nil {
		titleInfo, err := svc.ds.GetVPPTitleInfoByAdamIDAndPlatform(ctx, *p.VPPAdamID, fleet.MacOSPlatform)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get VPP title metadata by adam_id")
		}
		p.InstallSoftware = titleInfo
	}

	return nil
}

func (svc *Service) populatePolicyRunScript(ctx context.Context, p *fleet.Policy) error {
	if p.ScriptID == nil {
		return nil
	}
	scriptMetadata, err := svc.ds.Script(ctx, *p.ScriptID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get script metadata by id")
	}
	p.RunScript = &fleet.PolicyScript{ID: *p.ScriptID, Name: scriptMetadata.Name}
	return nil
}

func (svc *Service) newTeamPolicyPayloadToPolicyPayload(ctx context.Context, teamID uint, p fleet.NewTeamPolicyPayload) (fleet.PolicyPayload, error) {
	softwareInstallerID, vppAdamID, err := svc.getInstallerOrVPPAppForTitle(ctx, &teamID, p.SoftwareTitleID)
	if err != nil {
		return fleet.PolicyPayload{}, err
	}
	return fleet.PolicyPayload{
		QueryID:               p.QueryID,
		Name:                  p.Name,
		Query:                 p.Query,
		Critical:              p.Critical,
		Description:           p.Description,
		Resolution:            p.Resolution,
		Platform:              p.Platform,
		CalendarEventsEnabled: p.CalendarEventsEnabled,
		SoftwareInstallerID:   softwareInstallerID,
		VPPAdamID:             vppAdamID,
		ScriptID:              p.ScriptID,
	}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listTeamPoliciesRequest struct {
	TeamID                  uint                 `url:"team_id"`
	Opts                    fleet.ListOptions    `url:"list_options"`
	InheritedPage           uint                 `query:"inherited_page,optional"`
	InheritedPerPage        uint                 `query:"inherited_per_page,optional"`
	InheritedOrderDirection fleet.OrderDirection `query:"inherited_order_direction,optional"`
	InheritedOrderKey       string               `query:"inherited_order_key,optional"`
	MergeInherited          bool                 `query:"merge_inherited,optional"`
}

type listTeamPoliciesResponse struct {
	Policies          []*fleet.Policy `json:"policies,omitempty"`
	InheritedPolicies []*fleet.Policy `json:"inherited_policies,omitempty"`
	Err               error           `json:"error,omitempty"`
}

func (r listTeamPoliciesResponse) error() error { return r.Err }

func listTeamPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listTeamPoliciesRequest)

	inheritedListOptions := fleet.ListOptions{
		Page:           req.InheritedPage,
		PerPage:        req.InheritedPerPage,
		OrderDirection: req.InheritedOrderDirection,
		OrderKey:       req.InheritedOrderKey,
	}

	tmPols, inheritedPols, err := svc.ListTeamPolicies(ctx, req.TeamID, req.Opts, inheritedListOptions, req.MergeInherited)
	if err != nil {
		return listTeamPoliciesResponse{Err: err}, nil
	}
	return listTeamPoliciesResponse{Policies: tmPols, InheritedPolicies: inheritedPols}, nil
}

func (svc *Service) ListTeamPolicies(ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions, mergeInherited bool) (teamPolicies, inheritedPolicies []*fleet.Policy, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	if teamID > 0 {
		if _, err := svc.ds.Team(ctx, teamID); err != nil {
			return nil, nil, ctxerr.Wrapf(ctx, err, "loading team %d", teamID)
		}
	}

	if mergeInherited {
		policies, err := svc.ds.ListMergedTeamPolicies(ctx, teamID, opts)
		for i := range policies {
			if err := svc.populatePolicyInstallSoftware(ctx, policies[i]); err != nil {
				return nil, nil, ctxerr.Wrapf(ctx, err, "populate install_software for policy_id: %d", policies[i].ID)
			}
			if err := svc.populatePolicyRunScript(ctx, policies[i]); err != nil {
				return nil, nil, ctxerr.Wrapf(ctx, err, "populate run_script for policy_id: %d", policies[i].ID)
			}
		}
		return policies, nil, err
	}

	teamPolicies, inheritedPolicies, err = svc.ds.ListTeamPolicies(ctx, teamID, opts, iopts)
	if err != nil {
		return nil, nil, err
	}

	for i := range teamPolicies {
		if err := svc.populatePolicyInstallSoftware(ctx, teamPolicies[i]); err != nil {
			return nil, nil, ctxerr.Wrapf(ctx, err, "populate install_software for policy_id: %d", teamPolicies[i].ID)
		}
		if err := svc.populatePolicyRunScript(ctx, teamPolicies[i]); err != nil {
			return nil, nil, ctxerr.Wrapf(ctx, err, "populate run_script for policy_id: %d", teamPolicies[i].ID)
		}
	}

	return teamPolicies, inheritedPolicies, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Count
/////////////////////////////////////////////////////////////////////////////////

type countTeamPoliciesRequest struct {
	ListOptions    fleet.ListOptions `url:"list_options"`
	TeamID         uint              `url:"team_id"`
	MergeInherited bool              `query:"merge_inherited,optional"`
}

type countTeamPoliciesResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r countTeamPoliciesResponse) error() error { return r.Err }

func countTeamPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*countTeamPoliciesRequest)
	resp, err := svc.CountTeamPolicies(ctx, req.TeamID, req.ListOptions.MatchQuery, req.MergeInherited)
	if err != nil {
		return countTeamPoliciesResponse{Err: err}, nil
	}
	return countTeamPoliciesResponse{Count: resp}, nil
}

func (svc *Service) CountTeamPolicies(ctx context.Context, teamID uint, matchQuery string, mergeInherited bool) (int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionRead); err != nil {
		return 0, err
	}

	if teamID > 0 {
		if _, err := svc.ds.Team(ctx, teamID); err != nil {
			return 0, ctxerr.Wrapf(ctx, err, "loading team %d", teamID)
		}
	}

	if mergeInherited {
		return svc.ds.CountMergedTeamPolicies(ctx, teamID, matchQuery)
	}

	return svc.ds.CountPolicies(ctx, &teamID, matchQuery)
}

/////////////////////////////////////////////////////////////////////////////////
// Get by id
/////////////////////////////////////////////////////////////////////////////////

type getTeamPolicyByIDRequest struct {
	TeamID   uint `url:"team_id"`
	PolicyID uint `url:"policy_id"`
}

type getTeamPolicyByIDResponse struct {
	Policy *fleet.Policy `json:"policy"`
	Err    error         `json:"error,omitempty"`
}

func (r getTeamPolicyByIDResponse) error() error { return r.Err }

func getTeamPolicyByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getTeamPolicyByIDRequest)
	teamPolicy, err := svc.GetTeamPolicyByIDQueries(ctx, req.TeamID, req.PolicyID)
	if err != nil {
		return getTeamPolicyByIDResponse{Err: err}, nil
	}
	return getTeamPolicyByIDResponse{Policy: teamPolicy}, nil
}

func (svc Service) GetTeamPolicyByIDQueries(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	teamPolicy, err := svc.ds.TeamPolicy(ctx, teamID, policyID)
	if err != nil {
		return nil, err
	}

	if err := svc.populatePolicyInstallSoftware(ctx, teamPolicy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate install_software")
	}
	if err := svc.populatePolicyRunScript(ctx, teamPolicy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate run_script")
	}

	return teamPolicy, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete
/////////////////////////////////////////////////////////////////////////////////

type deleteTeamPoliciesRequest struct {
	TeamID uint   `url:"team_id"`
	IDs    []uint `json:"ids"`
}

type deleteTeamPoliciesResponse struct {
	Deleted []uint `json:"deleted,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r deleteTeamPoliciesResponse) error() error { return r.Err }

func deleteTeamPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteTeamPoliciesRequest)
	resp, err := svc.DeleteTeamPolicies(ctx, req.TeamID, req.IDs)
	if err != nil {
		return deleteTeamPoliciesResponse{Err: err}, nil
	}
	return deleteTeamPoliciesResponse{Deleted: resp}, nil
}

func (svc Service) DeleteTeamPolicies(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: ptr.Uint(teamID),
		},
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if teamID > 0 {
		if _, err := svc.ds.Team(ctx, teamID); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "loading team %d", teamID)
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}
	policiesByID, err := svc.ds.PoliciesByID(ctx, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting policies by ID")
	}

	for _, policy := range policiesByID {
		if t := policy.PolicyData.TeamID; t == nil || *t != teamID {
			return nil, authz.ForbiddenWithInternal(
				fmt.Sprintf("attempting to delete policy that does not belong to team %s", strconv.Itoa(int(teamID))),
				authz.UserFromContext(ctx),
				policy,
				fleet.ActionWrite,
			)
		}
	}

	deletedIDs, err := svc.ds.DeleteTeamPolicies(ctx, teamID, ids)
	if err != nil {
		return nil, err
	}

	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	for _, id := range deletedIDs {
		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeDeletedPolicy{
				ID:   id,
				Name: policiesByID[id].Name,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for policy deletion")
		}
	}

	return ids, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Modify
/////////////////////////////////////////////////////////////////////////////////

type modifyTeamPolicyRequest struct {
	TeamID   uint `url:"team_id"`
	PolicyID uint `url:"policy_id"`
	fleet.ModifyPolicyPayload
}

type modifyTeamPolicyResponse struct {
	Policy *fleet.Policy `json:"policy,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r modifyTeamPolicyResponse) error() error { return r.Err }

func modifyTeamPolicyEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyTeamPolicyRequest)
	resp, err := svc.ModifyTeamPolicy(ctx, req.TeamID, req.PolicyID, req.ModifyPolicyPayload)
	if err != nil {
		return modifyTeamPolicyResponse{Err: err}, nil
	}
	return modifyTeamPolicyResponse{Policy: resp}, nil
}

func (svc *Service) ModifyTeamPolicy(ctx context.Context, teamID uint, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	return svc.modifyPolicy(ctx, &teamID, id, p)
}

func checkTeamID(teamID *uint, policy *fleet.Policy) bool {
	return policy != nil && reflect.DeepEqual(teamID, policy.TeamID)
}

func (svc *Service) modifyPolicy(ctx context.Context, teamID *uint, id uint, p fleet.ModifyPolicyPayload) (*fleet.Policy, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			TeamID: teamID,
		},
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	policy, err := svc.ds.Policy(ctx, id)
	if err != nil {
		return nil, err
	}

	if ok := checkTeamID(teamID, policy); !ok {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "policy does not belong to team/global",
			InternalErr: fmt.Errorf("teamID: %+v, policy: %+v", teamID, policy),
		})
	}

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("policy payload verification: %s", err),
		})
	}

	var removeAllMemberships bool
	var removeStats bool
	if p.Name != nil {
		policy.Name = *p.Name
	}
	if p.Description != nil {
		policy.Description = *p.Description
	}
	if p.Query != nil {
		if policy.Query != *p.Query {
			removeAllMemberships = true
			removeStats = true
		}
		policy.Query = *p.Query
	}
	if p.Resolution != nil {
		policy.Resolution = p.Resolution
	}
	if p.Platform != nil {
		if policy.Platform != *p.Platform {
			removeStats = true
		}
		policy.Platform = *p.Platform
	}
	if p.Critical != nil {
		policy.Critical = *p.Critical
	}
	if p.CalendarEventsEnabled != nil {
		policy.CalendarEventsEnabled = *p.CalendarEventsEnabled
	}
	if removeStats {
		policy.FailingHostCount = 0
		policy.PassingHostCount = 0
	}
	if p.SoftwareTitleID.Set {
		softwareInstallerID, vppAdamID, err := svc.getInstallerOrVPPAppForTitle(ctx, teamID, &p.SoftwareTitleID.Value)
		if err != nil {
			return nil, err
		}
		// If the associated installer/app is changed (or it's set and the policy didn't have an associated installer/app)
		// then we clear the results of the policy so that automation can be triggered upon failure
		// (automation is currently triggered on the first failure or when it goes from passing to failure).
		if (softwareInstallerID != nil && (policy.SoftwareInstallerID == nil || *policy.SoftwareInstallerID != *softwareInstallerID)) ||
			(vppAdamID != nil && (policy.VPPAdamID == nil || *policy.VPPAdamID != *vppAdamID)) {
			removeAllMemberships = true
			removeStats = true
		}
		policy.SoftwareInstallerID = softwareInstallerID
		policy.VPPAdamID = vppAdamID
	}
	if p.ScriptID.Set { // indicates that script ID is changing, but might be to 0 to remove
		// If the associated script is changed (or it's set and the policy didn't have an associated script)
		// then we clear the results of the policy so that automation can be triggered upon failure
		// (automation is currently triggered on the first failure or when it goes from passing to failure).
		if p.ScriptID.Value != 0 && (policy.ScriptID == nil || *policy.ScriptID != p.ScriptID.Value) {
			removeAllMemberships = true
			removeStats = true
		}

		if p.ScriptID.Value == 0 {
			policy.ScriptID = nil
		} else {
			policy.ScriptID = &p.ScriptID.Value
		}
	}

	logging.WithExtras(ctx, "name", policy.Name, "sql", policy.Query)

	err = svc.ds.SavePolicy(ctx, policy, removeAllMemberships, removeStats)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving policy")
	}

	if err := svc.populatePolicyInstallSoftware(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate install_software")
	}
	if err := svc.populatePolicyRunScript(ctx, policy); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "populate run_script")
	}

	// Note: Issue #4191 proposes that we move to SQL transactions for actions so that we can
	// rollback an action in the event of an error writing the associated activity
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedPolicy{
			ID:   policy.ID,
			Name: policy.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for policy modification")
	}

	return policy, nil
}

func (svc *Service) getInstallerOrVPPAppForTitle(ctx context.Context, teamID *uint, softwareTitleID *uint) (installerID *uint, vppAdamID *string, err error) {
	if softwareTitleID == nil {
		return nil, nil, nil
	}

	// If *p.SoftwareTitleID with value 0 is used to unset the current installer from the policy.
	if *softwareTitleID == 0 {
		return nil, nil, nil
	}

	if teamID == nil {
		return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: "software_title_id cannot be set on global policies",
		})
	}

	softwareTitle, err := svc.SoftwareTitleByID(ctx, *softwareTitleID, teamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("software_title_id %d on team_id %d not found", *softwareTitleID, *teamID),
			})
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "software title by id")
	}
	if softwareTitle.AppStoreApp != nil {
		if softwareTitle.AppStoreApp.Platform != fleet.MacOSPlatform {
			return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf(
					"software_title_id %d on team_id %d is assocated to an iOS or iPadOS VPP app, only software installers or macOS VPP apps are supported",
					*softwareTitleID,
					*teamID,
				),
			})
		}

		return nil, &softwareTitle.AppStoreApp.AdamID, nil
	}
	if softwareTitle.SoftwarePackage == nil {
		return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("software_title_id %d on team_id %d does not have associated package", *softwareTitleID, *teamID),
		})
	}

	// At this point we assume *softwareTitle.SoftwarePackage.TeamID == *teamID,
	// because SoftwareTitleByID above receives the teamID.
	return ptr.Uint(softwareTitle.SoftwarePackage.InstallerID), nil, nil
}
