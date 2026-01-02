package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Create Label
////////////////////////////////////////////////////////////////////////////////

type createLabelRequest struct {
	fleet.LabelPayload
}

type createLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r createLabelResponse) Error() error { return r.Err }

func createLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createLabelRequest)

	label, hostIDs, err := svc.NewLabel(ctx, req.LabelPayload)
	if err != nil {
		return createLabelResponse{Err: err}, nil
	}

	labelResp, err := labelResponseForLabel(label, hostIDs)
	if err != nil {
		return createLabelResponse{Err: err}, nil
	}

	return createLabelResponse{Label: *labelResp}, nil
}

func (svc *Service) NewLabel(ctx context.Context, p fleet.LabelPayload) (*fleet.Label, []uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionCreate); err != nil {
		return nil, nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}

	if len(p.Hosts) > 0 && len(p.HostIDs) > 0 {
		return nil, nil, fleet.NewInvalidArgumentError("hosts", `Only one of either "hosts" or "host_ids" can be included in the request.`)
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	label := &fleet.Label{
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		AuthorID:            ptr.Uint(vc.UserID()),
	}

	if p.Name == "" {
		return nil, nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	label.Name = p.Name

	if p.Criteria != nil {
		if p.Query != "" || (len(p.Hosts) > 0 || len(p.HostIDs) > 0) {
			return nil, nil, fleet.NewInvalidArgumentError("criteria", `Only one of "criteria", "query" or "hosts/host_ids" can be included in the request.`)
		}
		label.LabelMembershipType = fleet.LabelMembershipTypeHostVitals
		labelCriteriaJson, err := json.Marshal(p.Criteria)
		if err != nil {
			return nil, nil, fleet.NewInvalidArgumentError("criteria", fmt.Sprintf("invalid criteria: %s", err.Error()))
		}
		label.HostVitalsCriteria = ptr.RawMessage(json.RawMessage(labelCriteriaJson))
		// Attempt to calculate a query from the criteria.
		_, _, err = label.CalculateHostVitalsQuery()
		if err != nil {
			return nil, nil, fleet.NewInvalidArgumentError("criteria", fmt.Sprintf("invalid criteria: %s", err.Error()))
		}
	} else {
		if p.Query != "" && (len(p.Hosts) > 0 || len(p.HostIDs) > 0) {
			return nil, nil, fleet.NewInvalidArgumentError("query", `Only one of "criteria", "query" or "hosts/host_ids" can be included in the request.`)
		}
		label.Query = p.Query
		if p.Query == "" {
			label.LabelMembershipType = fleet.LabelMembershipTypeManual
		}
	}

	label.Platform = p.Platform
	label.Description = p.Description

	for name := range fleet.ReservedLabelNames() {
		if label.Name == name {
			return nil, nil, fleet.NewInvalidArgumentError("name", fmt.Sprintf("cannot add label '%s' because it conflicts with the name of a built-in label", name))
		}
	}

	// first create the new label, which will fail if the name is not unique
	var err error
	label, err = svc.ds.NewLabel(ctx, label)
	if err != nil {
		return nil, nil, err
	}

	if label.LabelMembershipType == fleet.LabelMembershipTypeManual {
		hostIDs := p.HostIDs
		if len(p.Hosts) > 0 {
			hostIDs, err = svc.ds.HostIDsByIdentifier(ctx, filter, p.Hosts)
			if err != nil {
				return nil, nil, err
			}
		}
		return svc.ds.UpdateLabelMembershipByHostIDs(ctx, *label, hostIDs, filter)
	}
	return label, nil, nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify Label
////////////////////////////////////////////////////////////////////////////////

type modifyLabelRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.ModifyLabelPayload
}

type modifyLabelResponse struct {
	Label labelWithTeamNameResponse `json:"label"`
	Err   error                     `json:"error,omitempty"`
}

func (r modifyLabelResponse) Error() error { return r.Err }

func modifyLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*modifyLabelRequest)
	label, hostIDs, err := svc.ModifyLabel(ctx, req.ID, req.ModifyLabelPayload)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	labelResp, err := labelResponseForLabelWithTeamName(label, hostIDs)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	return modifyLabelResponse{Label: *labelResp}, err
}

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload fleet.ModifyLabelPayload) (*fleet.LabelWithTeamName, []uint, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		svc.SkipAuth(ctx)
		return nil, nil, fleet.ErrNoContext
	}

	if len(payload.Hosts) > 0 && len(payload.HostIDs) > 0 {
		svc.SkipAuth(ctx)
		return nil, nil, fleet.NewInvalidArgumentError("hosts", `Only one of either "hosts" or "host_ids" can be included in the request.`)
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	// DB query will filter labels the user can't see; auth check filters labels the user can't write
	label, _, err := svc.ds.Label(ctx, id, filter)
	if err != nil {
		// If we get a retrieval error, 403-wrap it if a user can't write global labels so we don't leak info
		if authErr := svc.authz.Authorize(ctx, fleet.Label{}, fleet.ActionWrite); authErr != nil {
			return nil, nil, authErr
		}
		return nil, nil, err
	}
	if err := svc.authz.Authorize(ctx, label, fleet.ActionWrite); err != nil {
		return nil, nil, err
	}

	if label.LabelType == fleet.LabelTypeBuiltIn {
		return nil, nil, fleet.NewInvalidArgumentError("label_type", fmt.Sprintf("cannot modify built-in label '%s'", label.Name))
	}
	if payload.Name != nil {
		// Check if the new name is a reserved label name
		for name := range fleet.ReservedLabelNames() {
			if *payload.Name == name {
				return nil, nil, fleet.NewInvalidArgumentError("name", fmt.Sprintf("cannot rename label to '%s' because it conflicts with the name of a built-in label", name))
			}
		}
		label.Name = *payload.Name
	}
	if payload.Description != nil {
		label.Description = *payload.Description
	}

	hostIDs := payload.HostIDs
	if len(payload.Hosts) > 0 {
		// If hosts were provided, convert them to IDs.
		hostIDs, err = svc.ds.HostIDsByIdentifier(ctx, filter, payload.Hosts)
		if err != nil {
			return nil, nil, err
		}
	} else if payload.Hosts != nil {
		// If an empry list was provided, create an empty list of IDs
		// so that we can remove all hosts from the label.
		hostIDs = make([]uint, 0)
	}

	if len(hostIDs) > 0 && label.LabelMembershipType != fleet.LabelMembershipTypeManual {
		return nil, nil, fleet.NewInvalidArgumentError("hosts", "cannot provide a list of hosts for a dynamic label")
	}

	if hostIDs != nil {
		if _, _, err := svc.ds.UpdateLabelMembershipByHostIDs(ctx, label.Label, hostIDs, filter); err != nil {
			return nil, nil, err
		}
	}

	return svc.ds.SaveLabel(ctx, &label.Label, filter)
}

////////////////////////////////////////////////////////////////////////////////
// Get Label
////////////////////////////////////////////////////////////////////////////////

type getLabelRequest struct {
	ID uint `url:"id"`
}

type labelWithTeamNameResponse struct {
	fleet.LabelWithTeamName
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
	HostIDs     []uint `json:"host_ids,omitempty"`
}

type labelResponse struct {
	fleet.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
	HostIDs     []uint `json:"host_ids,omitempty"`
}

type getLabelResponse struct {
	Label labelWithTeamNameResponse `json:"label"`
	Err   error                     `json:"error,omitempty"`
}

func (r getLabelResponse) Error() error { return r.Err }

func getLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getLabelRequest)
	label, hostIDs, err := svc.GetLabel(ctx, req.ID)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	resp, err := labelResponseForLabelWithTeamName(label, hostIDs)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	return getLabelResponse{Label: *resp}, nil
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*fleet.LabelWithTeamName, []uint, error) {
	// authz intentionally casts a wide net here; we filter unauthorized labels out at the data store level
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.Label(ctx, id, filter)
}

////////////////////////////////////////////////////////////////////////////////
// List Labels
////////////////////////////////////////////////////////////////////////////////

type listLabelsRequest struct {
	ListOptions       fleet.ListOptions `url:"list_options"`
	TeamID            *string           `query:"team_id,optional"` // string because it's an int or "global"
	IncludeHostCounts *bool             `query:"include_host_counts,optional"`
}

type listLabelsResponse struct {
	Labels []labelResponse `json:"labels"`
	Err    error           `json:"error,omitempty"`
}

func (r listLabelsResponse) Error() error { return r.Err }

func listLabelsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listLabelsRequest)

	includeHostCounts := true
	if req.IncludeHostCounts != nil {
		includeHostCounts = *req.IncludeHostCounts
	}

	labels, err := svc.ListLabels(ctx, req.ListOptions, getTeamIDOrZeroForGlobal(req.TeamID), includeHostCounts)
	if err != nil {
		return listLabelsResponse{Err: err}, nil
	}

	resp := listLabelsResponse{}
	for _, label := range labels {
		labelResp, err := labelResponseForLabel(label, nil)
		if err != nil {
			return listLabelsResponse{Err: err}, nil
		}
		resp.Labels = append(resp.Labels, *labelResp)
	}
	return resp, nil
}

func getTeamIDOrZeroForGlobal(stringID *string) *uint {
	if stringID == nil || *stringID == "" {
		return nil
	}

	if *stringID == "global" {
		return ptr.Uint(0)
	}

	if parsedTeamID, err := strconv.ParseUint(*stringID, 10, 32); err == nil {
		return ptr.Uint(uint(parsedTeamID))
	}

	return nil
}

func (svc *Service) ListLabels(ctx context.Context, opt fleet.ListOptions, teamID *uint, includeHostCounts bool) ([]*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	if !license.IsPremium(ctx) && teamID != nil && *teamID > 0 {
		return nil, fleet.ErrMissingLicense
	}

	// TODO(mna): ListLabels doesn't currently return the hostIDs members of the
	// label, the quick approach would be an N+1 queries endpoint. Leaving like
	// that for now because we're in a hurry before merge freeze but the solution
	// would probably be to do it in 2 queries : grab all label IDs from the
	// list, then select hostID+labelID tuples in one query (where labelID IN
	// <list of ids>)and fill the hostIDs per label.
	return svc.ds.ListLabels(ctx, fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID}, opt, includeHostCounts)
}

func labelResponseForLabel(label *fleet.Label, hostIDs []uint) (*labelResponse, error) {
	return &labelResponse{
		Label:       *label,
		DisplayText: label.Name,
		Count:       label.HostCount,
		HostIDs:     hostIDs,
	}, nil
}

func labelResponseForLabelWithTeamName(label *fleet.LabelWithTeamName, hostIDs []uint) (*labelWithTeamNameResponse, error) {
	return &labelWithTeamNameResponse{
		LabelWithTeamName: *label,
		DisplayText:       label.Name,
		Count:             label.HostCount,
		HostIDs:           hostIDs,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Labels Summary
////////////////////////////////////////////////////////////////////////////////

type getLabelsSummaryRequest struct {
	TeamID *string `query:"team_id,optional"` // string because it's an int or "global"
}

type getLabelsSummaryResponse struct {
	Labels []*fleet.LabelSummary `json:"labels"`
	Err    error                 `json:"error,omitempty"`
}

func (r getLabelsSummaryResponse) Error() error { return r.Err }

func getLabelsSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getLabelsSummaryRequest)

	labels, err := svc.LabelsSummary(ctx, getTeamIDOrZeroForGlobal(req.TeamID))
	if err != nil {
		return getLabelsSummaryResponse{Err: err}, nil
	}
	return getLabelsSummaryResponse{Labels: labels}, nil
}

func (svc *Service) LabelsSummary(ctx context.Context, teamID *uint) ([]*fleet.LabelSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	if !license.IsPremium(ctx) && teamID != nil && *teamID > 0 {
		return nil, fleet.ErrMissingLicense
	}

	return svc.ds.LabelsSummary(ctx, fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID})
}

////////////////////////////////////////////////////////////////////////////////
// List Hosts in Label
////////////////////////////////////////////////////////////////////////////////

type listHostsInLabelRequest struct {
	ID          uint                  `url:"id"`
	ListOptions fleet.HostListOptions `url:"host_options"`
}

func listHostsInLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostsInLabelRequest)
	hosts, err := svc.ListHostsInLabel(ctx, req.ID, req.ListOptions)
	if err != nil {
		return listLabelsResponse{Err: err}, nil
	}

	var mdmSolution *fleet.MDMSolution
	if req.ListOptions.MDMIDFilter != nil {
		var err error
		mdmSolution, err = svc.GetMDMSolution(ctx, *req.ListOptions.MDMIDFilter)
		if err != nil && !fleet.IsNotFound(err) { // ignore not found, just return nil for the MDM solution in that case
			return listHostsResponse{Err: err}, nil
		}
	}

	hostResponses := make([]fleet.HostResponse, len(hosts))
	for i, host := range hosts {
		h := fleet.HostResponseForHost(ctx, svc, host)
		hostResponses[i] = *h
	}
	return listHostsResponse{Hosts: hostResponses, MDMSolution: mdmSolution}, nil
}

func (svc *Service) ListHostsInLabel(ctx context.Context, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	hosts, err := svc.ds.ListHostsInLabel(ctx, filter, lid, opt)
	if err != nil {
		return nil, err
	}

	premiumLicense := license.IsPremium(ctx)
	// If issues are enabled, we need to remove the critical vulnerabilities count for non-premium license.
	// If issues are disabled, we need to explicitly set the critical vulnerabilities count to 0 for premium license.
	if !opt.DisableIssues && !premiumLicense {
		// Remove critical vulnerabilities count if not premium license
		for _, host := range hosts {
			host.HostIssues.CriticalVulnerabilitiesCount = nil
		}
	} else if opt.DisableIssues && premiumLicense {
		var zero uint64
		for _, host := range hosts {
			host.HostIssues.CriticalVulnerabilitiesCount = &zero
		}
	}
	return hosts, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Label
////////////////////////////////////////////////////////////////////////////////

type deleteLabelRequest struct {
	Name string `url:"name"`
}

type deleteLabelResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteLabelResponse) Error() error { return r.Err }

func deleteLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteLabelRequest)
	err := svc.DeleteLabel(ctx, req.Name)
	if err != nil {
		return deleteLabelResponse{Err: err}, nil
	}
	return deleteLabelResponse{}, nil
}

func (svc *Service) DeleteLabel(ctx context.Context, name string) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		svc.SkipAuth(ctx)
		return fleet.ErrNoContext
	}

	// check if the label is a built-in label
	for n := range fleet.ReservedLabelNames() {
		if n == name {
			svc.SkipAuth(ctx)
			return fleet.NewInvalidArgumentError("name", fmt.Sprintf("cannot delete built-in label '%s'", name))
		}
	}

	filter := fleet.TeamFilter{User: vc.User}

	// need to grab the label first to see if we have permission to delete it;
	// if the label doesn't exist global users will see the true 404, other users will get a 403
	label, err := svc.ds.LabelByName(ctx, name, filter)
	if err != nil {
		if authError := svc.authz.Authorize(ctx, fleet.Label{}, fleet.ActionWrite); authError != nil {
			return authError
		}
		return err
	}
	if err := svc.authz.Authorize(ctx, label, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteLabel(ctx, name, filter)
}

////////////////////////////////////////////////////////////////////////////////
// Delete Label By ID
////////////////////////////////////////////////////////////////////////////////

type deleteLabelByIDRequest struct {
	ID uint `url:"id"`
}

type deleteLabelByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteLabelByIDResponse) Error() error { return r.Err }

func deleteLabelByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteLabelByIDRequest)
	err := svc.DeleteLabelByID(ctx, req.ID)
	if err != nil {
		return deleteLabelByIDResponse{Err: err}, nil
	}
	return deleteLabelByIDResponse{}, nil
}

func (svc *Service) DeleteLabelByID(ctx context.Context, id uint) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		svc.SkipAuth(ctx)
		return fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	// need to grab the label first to see if we have permission to delete it;
	// if the label doesn't exist global users will see the true 404, other users will get a 403
	label, _, err := svc.ds.Label(ctx, id, filter)
	if err != nil {
		// If we get a retrieval error, 403-wrap it if a user can't write global labels so we don't leak info
		if authErr := svc.authz.Authorize(ctx, fleet.Label{}, fleet.ActionWrite); authErr != nil {
			return authErr
		}
		return err
	}
	if err := svc.authz.Authorize(ctx, label, fleet.ActionWrite); err != nil {
		return err
	}

	if label.LabelType == fleet.LabelTypeBuiltIn {
		return fleet.NewInvalidArgumentError("label_type", fmt.Sprintf("cannot delete built-in label '%s'", label.Name))
	}
	for name := range fleet.ReservedLabelNames() {
		if label.Name == name {
			return fleet.NewInvalidArgumentError("name", fmt.Sprintf("cannot delete built-in label '%s'", label.Name))
		}
	}

	return svc.ds.DeleteLabel(ctx, label.Name, filter)
}

////////////////////////////////////////////////////////////////////////////////
// Apply Label Specs
////////////////////////////////////////////////////////////////////////////////

type applyLabelSpecsRequest struct {
	Specs       []*fleet.LabelSpec `json:"specs"`
	TeamID      *uint              `json:"-" query:"team_id,optional"`
	NamesToMove []string           `json:"names_to_move,omitempty"`
}

type applyLabelSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyLabelSpecsResponse) Error() error { return r.Err }

func applyLabelSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyLabelSpecsRequest)
	err := svc.ApplyLabelSpecs(ctx, req.Specs, req.TeamID, req.NamesToMove)
	if err != nil {
		return applyLabelSpecsResponse{Err: err}, nil
	}
	return applyLabelSpecsResponse{}, nil
}

func (svc *Service) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec, teamID *uint, namesToMove []string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	user, ok := viewer.FromContext(ctx)
	if !ok || user.User == nil {
		return fleet.ErrNoContext
	}
	if !license.IsPremium(ctx) && teamID != nil && *teamID > 0 {
		return fleet.ErrMissingLicense
	}

	regularSpecs := make([]*fleet.LabelSpec, 0, len(specs))
	var builtInSpecs []*fleet.LabelSpec
	var builtInSpecNames []string

	var specLabelNamesNeedingMoving []string // should match namesToMove once specs have been checked

	for _, spec := range specs {
		if spec.LabelMembershipType == fleet.LabelMembershipTypeDynamic && len(spec.Hosts) > 0 {
			return fleet.NewUserMessageError(
				ctxerr.Errorf(ctx, "label %s is declared as dynamic but contains `hosts` key", spec.Name), http.StatusUnprocessableEntity,
			)
		}
		if spec.LabelMembershipType == fleet.LabelMembershipTypeManual && spec.Hosts == nil {
			// Hosts list doesn't need to contain anything, but it should at least not be nil.
			return fleet.NewUserMessageError(
				ctxerr.Errorf(ctx, "label %s is declared as manual but contains no `hosts key`", spec.Name), http.StatusUnprocessableEntity,
			)
		}
		if spec.LabelMembershipType == fleet.LabelMembershipTypeHostVitals && spec.HostVitalsCriteria == nil {
			// Criteria is required for host vitals labels.
			return fleet.NewUserMessageError(
				ctxerr.Errorf(ctx, "label %s is declared as host vitals but contains no `criteria` key", spec.Name), http.StatusUnprocessableEntity,
			)
		}
		if spec.LabelType == fleet.LabelTypeBuiltIn {
			// We allow specs to contain built-in labels as long as they are not being modified.
			// This allows the user to do the following workflow without manually removing built-in labels:
			// 1. fleetctl get labels --yaml > labels.yml
			// 2. (Optional) Edit labels.yml
			// 3. fleetctl apply -f labels.yml
			builtInSpecs = append(builtInSpecs, spec)
			builtInSpecNames = append(builtInSpecNames, spec.Name)
			continue
		}
		for name := range fleet.ReservedLabelNames() {
			if spec.Name == name {
				return fleet.NewUserMessageError(
					ctxerr.Errorf(
						ctx,
						"cannot add label '%s' because it conflicts with the name of a built-in label",
						name,
					), http.StatusUnprocessableEntity)
			}
		}

		if slices.Contains(namesToMove, spec.Name) {
			specLabelNamesNeedingMoving = append(specLabelNamesNeedingMoving, spec.Name)
		}

		// make sure we're only upserting labels on the team we specified; individual spec teams aren't used on writes
		spec.TeamID = teamID
		regularSpecs = append(regularSpecs, spec)
	}

	if len(specLabelNamesNeedingMoving) != len(namesToMove) {
		return fleet.NewUserMessageError(
			ctxerr.New(ctx, "label names to move list was not a subset of specified labels"),
			http.StatusConflict,
		)
	}

	// If built-in labels have been provided, ensure that they are not attempted to be modified
	if len(builtInSpecs) > 0 {
		labelMap, err := svc.ds.LabelsByName(ctx, builtInSpecNames, fleet.TeamFilter{}) // built-in labels are all global
		if err != nil {
			return err
		}
		for _, spec := range builtInSpecs {
			label, ok := labelMap[spec.Name]
			if !ok ||
				label.Description != spec.Description ||
				label.Query != spec.Query ||
				label.Platform != spec.Platform ||
				label.LabelType != fleet.LabelTypeBuiltIn ||
				label.LabelMembershipType != spec.LabelMembershipType {
				return fleet.NewUserMessageError(
					ctxerr.Errorf(ctx, "cannot modify or add built-in label '%s'", spec.Name), http.StatusUnprocessableEntity,
				)
			}
		}
	}
	if len(regularSpecs) == 0 {
		return nil
	}

	if err := svc.ds.SetAsideLabels(ctx, teamID, namesToMove, *user.User); err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up conflicting other team labels")
	}

	return svc.ds.ApplyLabelSpecsWithAuthor(ctx, regularSpecs, ptr.Uint(user.UserID()))
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Specs
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecsResponse struct {
	Specs []*fleet.LabelSpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getLabelSpecsResponse) Error() error { return r.Err }

type getLabelSpecsRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

func getLabelSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getLabelSpecsRequest)
	specs, err := svc.GetLabelSpecs(ctx, req.TeamID)
	if err != nil {
		return getLabelSpecsResponse{Err: err}, nil
	}
	return getLabelSpecsResponse{Specs: specs}, nil
}

func (svc *Service) GetLabelSpecs(ctx context.Context, teamID *uint) ([]*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if !license.IsPremium(ctx) && teamID != nil && *teamID > 0 {
		return nil, fleet.ErrMissingLicense
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	return svc.ds.GetLabelSpecs(ctx, fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID})
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Spec
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecResponse struct {
	Spec *fleet.LabelSpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getLabelSpecResponse) Error() error { return r.Err }

func getLabelSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getGenericSpecRequest)
	spec, err := svc.GetLabelSpec(ctx, req.Name)
	if err != nil {
		return getLabelSpecResponse{Err: err}, nil
	}
	return getLabelSpecResponse{Spec: spec}, nil
}

func (svc *Service) GetLabelSpec(ctx context.Context, name string) (*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	return svc.ds.GetLabelSpec(ctx, fleet.TeamFilter{User: vc.User, IncludeObserver: true}, name)
}

func (svc *Service) BatchValidateLabels(ctx context.Context, teamID *uint, labelNames []string) (map[string]fleet.LabelIdent, error) {
	if authctx, ok := authz_ctx.FromContext(ctx); !ok {
		return nil, fleet.NewAuthRequiredError("batch validate labels: missing authorization context")
	} else if !authctx.Checked() {
		return nil, fleet.NewAuthRequiredError("batch validate labels: method requires previous authorization")
	}

	if len(labelNames) == 0 {
		return nil, nil
	}

	uniqueNames := server.RemoveDuplicatesFromSlice(labelNames)

	labels, err := svc.ds.LabelIDsByName(ctx, uniqueNames, fleet.TeamFilter{User: authz.UserFromContext(ctx)})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	if len(labels) != len(uniqueNames) {
		return nil, &fleet.BadRequestError{
			Message:     "some or all the labels provided don't exist",
			InternalErr: fmt.Errorf("names provided: %v", labelNames),
		}
	}

	if err := verifyLabelsToAssociate(ctx, svc.ds, teamID, labelNames, authz.UserFromContext(ctx)); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "verify labels to associate")
	}

	byName := make(map[string]fleet.LabelIdent, len(labels))
	for labelName, labelID := range labels {
		byName[labelName] = fleet.LabelIdent{
			LabelName: labelName,
			LabelID:   labelID,
		}
	}
	return byName, nil
}
