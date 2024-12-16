package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (r createLabelResponse) error() error { return r.Err }

func createLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	label := &fleet.Label{
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	}

	if p.Name == "" {
		return nil, nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	label.Name = p.Name

	if p.Query != "" && len(p.Hosts) > 0 {
		return nil, nil, fleet.NewInvalidArgumentError("query", `Only one of either "query" or "hosts" can be included in the request.`)
	}
	label.Query = p.Query
	if p.Query == "" {
		label.LabelMembershipType = fleet.LabelMembershipTypeManual
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

	// Next, if membership type is manual, use ApplyLabelSpecs to create label
	// memberships. Must resolve the host identifiers to hostname so that
	// ApplySpecs can be used.
	var hostIDs []uint
	if label.LabelMembershipType == fleet.LabelMembershipTypeManual {
		spec := fleet.LabelSpec{
			Name:                label.Name,
			Description:         label.Description,
			Query:               label.Query,
			Platform:            label.Platform,
			LabelType:           label.LabelType,
			LabelMembershipType: label.LabelMembershipType,
		}
		hostnames, err := svc.ds.HostnamesByIdentifiers(ctx, p.Hosts)
		if err != nil {
			return nil, nil, err
		}
		spec.Hosts = hostnames
		if err := svc.ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{&spec}); err != nil {
			return nil, nil, err
		}

		// must reload it to get the host IDs, refresh its count
		ctx = ctxdb.RequirePrimary(ctx, true)
		label, hostIDs, err = svc.ds.Label(ctx, label.ID, filter)
		if err != nil {
			return nil, nil, err
		}
	}
	return label, hostIDs, nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify Label
////////////////////////////////////////////////////////////////////////////////

type modifyLabelRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.ModifyLabelPayload
}

type modifyLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r modifyLabelResponse) error() error { return r.Err }

func modifyLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyLabelRequest)
	label, hostIDs, err := svc.ModifyLabel(ctx, req.ID, req.ModifyLabelPayload)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	labelResp, err := labelResponseForLabel(label, hostIDs)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	return modifyLabelResponse{Label: *labelResp}, err
}

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload fleet.ModifyLabelPayload) (*fleet.Label, []uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	label, _, err := svc.ds.Label(ctx, id, filter)
	if err != nil {
		return nil, nil, err
	}
	if label.LabelType == fleet.LabelTypeBuiltIn {
		return nil, nil, fleet.NewInvalidArgumentError("label_type", fmt.Sprintf("cannot modify built-in label '%s'", label.Name))
	}
	originalLabelName := label.Name
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
	if len(payload.Hosts) > 0 && label.LabelMembershipType != fleet.LabelMembershipTypeManual {
		return nil, nil, fleet.NewInvalidArgumentError("hosts", "cannot provide a list of hosts for a dynamic label")
	}

	// if membership type is manual and the Hosts membership is provided, must
	// use ApplyLabelSpecs (as SaveLabel does not update label memberships),
	// otherwise SaveLabel works for dynamic membership. Must resolve the host
	// identifiers to hostname so that ApplySpecs can be used (it expects only
	// hostnames).
	if label.LabelMembershipType == fleet.LabelMembershipTypeManual && payload.Hosts != nil {
		spec := fleet.LabelSpec{
			Name:                originalLabelName,
			Description:         label.Description,
			Query:               label.Query,
			Platform:            label.Platform,
			LabelType:           label.LabelType,
			LabelMembershipType: label.LabelMembershipType,
		}
		hostnames, err := svc.ds.HostnamesByIdentifiers(ctx, payload.Hosts)
		if err != nil {
			return nil, nil, err
		}
		spec.Hosts = hostnames
		// Note: ApplyLabelSpecs cannot update label name since it uses the name as a key.
		// So, we must handle it later.
		if err := svc.ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{&spec}); err != nil {
			return nil, nil, err
		}
		// If the label name has changed, we must update it.
		if originalLabelName != label.Name {
			return svc.ds.SaveLabel(ctx, label, filter)
		}
		// Otherwise, simply reload label to get the host counts information
		ctx = ctxdb.RequirePrimary(ctx, true)
		return svc.ds.Label(ctx, id, filter)
	}
	return svc.ds.SaveLabel(ctx, label, filter)
}

////////////////////////////////////////////////////////////////////////////////
// Get Label
////////////////////////////////////////////////////////////////////////////////

type getLabelRequest struct {
	ID uint `url:"id"`
}

type labelResponse struct {
	fleet.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
	HostIDs     []uint `json:"host_ids,omitempty"`
}

type getLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r getLabelResponse) error() error { return r.Err }

func getLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getLabelRequest)
	label, hostIDs, err := svc.GetLabel(ctx, req.ID)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	resp, err := labelResponseForLabel(label, hostIDs)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	return getLabelResponse{Label: *resp}, nil
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*fleet.Label, []uint, error) {
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
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listLabelsResponse struct {
	Labels []labelResponse `json:"labels"`
	Err    error           `json:"error,omitempty"`
}

func (r listLabelsResponse) error() error { return r.Err }

func listLabelsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listLabelsRequest)

	labels, err := svc.ListLabels(ctx, req.ListOptions)
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

func (svc *Service) ListLabels(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	// TODO(mna): ListLabels doesn't currently return the hostIDs members of the
	// label, the quick approach would be an N+1 queries endpoint. Leaving like
	// that for now because we're in a hurry before merge freeze but the solution
	// would probably be to do it in 2 queries : grab all label IDs from the
	// list, then select hostID+labelID tuples in one query (where labelID IN
	// <list of ids>)and fill the hostIDs per label.
	return svc.ds.ListLabels(ctx, filter, opt)
}

func labelResponseForLabel(label *fleet.Label, hostIDs []uint) (*labelResponse, error) {
	return &labelResponse{
		Label:       *label,
		DisplayText: label.Name,
		Count:       label.HostCount,
		HostIDs:     hostIDs,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Labels Summary
////////////////////////////////////////////////////////////////////////////////

type getLabelsSummaryResponse struct {
	Labels []*fleet.LabelSummary `json:"labels"`
	Err    error                 `json:"error,omitempty"`
}

func (r getLabelsSummaryResponse) error() error { return r.Err }

func getLabelsSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	labels, err := svc.LabelsSummary(ctx)
	if err != nil {
		return getLabelsSummaryResponse{Err: err}, nil
	}
	return getLabelsSummaryResponse{Labels: labels}, nil
}

func (svc *Service) LabelsSummary(ctx context.Context) ([]*fleet.LabelSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.LabelsSummary(ctx)
}

////////////////////////////////////////////////////////////////////////////////
// List Hosts in Label
////////////////////////////////////////////////////////////////////////////////

type listHostsInLabelRequest struct {
	ID          uint                  `url:"id"`
	ListOptions fleet.HostListOptions `url:"host_options"`
}

func listHostsInLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

func (r deleteLabelResponse) error() error { return r.Err }

func deleteLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteLabelRequest)
	err := svc.DeleteLabel(ctx, req.Name)
	if err != nil {
		return deleteLabelResponse{Err: err}, nil
	}
	return deleteLabelResponse{}, nil
}

func (svc *Service) DeleteLabel(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	// check if the label is a built-in label
	for n := range fleet.ReservedLabelNames() {
		if n == name {
			return fleet.NewInvalidArgumentError("name", fmt.Sprintf("cannot delete built-in label '%s'", name))
		}
	}

	return svc.ds.DeleteLabel(ctx, name)
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

func (r deleteLabelByIDResponse) error() error { return r.Err }

func deleteLabelByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteLabelByIDRequest)
	err := svc.DeleteLabelByID(ctx, req.ID)
	if err != nil {
		return deleteLabelByIDResponse{Err: err}, nil
	}
	return deleteLabelByIDResponse{}, nil
}

func (svc *Service) DeleteLabelByID(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	label, _, err := svc.ds.Label(ctx, id, filter)
	if err != nil {
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

	return svc.ds.DeleteLabel(ctx, label.Name)
}

////////////////////////////////////////////////////////////////////////////////
// Apply Label Specs
////////////////////////////////////////////////////////////////////////////////

type applyLabelSpecsRequest struct {
	Specs []*fleet.LabelSpec `json:"specs"`
}

type applyLabelSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyLabelSpecsResponse) error() error { return r.Err }

func applyLabelSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*applyLabelSpecsRequest)
	err := svc.ApplyLabelSpecs(ctx, req.Specs)
	if err != nil {
		return applyLabelSpecsResponse{Err: err}, nil
	}
	return applyLabelSpecsResponse{}, nil
}

func (svc *Service) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return err
	}

	regularSpecs := make([]*fleet.LabelSpec, 0, len(specs))
	var builtInSpecs []*fleet.LabelSpec
	var builtInSpecNames []string
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
				return fleet.NewUserMessageError(ctxerr.Errorf(ctx, "cannot modify built-in label '%s'", name), http.StatusUnprocessableEntity)
			}
		}
		regularSpecs = append(regularSpecs, spec)
	}

	// If built-in labels have been provided, ensure that they are not attempted to be modified
	if len(builtInSpecs) > 0 {
		labelMap, err := svc.ds.LabelsByName(ctx, builtInSpecNames)
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
	return svc.ds.ApplyLabelSpecs(ctx, regularSpecs)
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Specs
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecsResponse struct {
	Specs []*fleet.LabelSpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getLabelSpecsResponse) error() error { return r.Err }

func getLabelSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	specs, err := svc.GetLabelSpecs(ctx)
	if err != nil {
		return getLabelSpecsResponse{Err: err}, nil
	}
	return getLabelSpecsResponse{Specs: specs}, nil
}

func (svc *Service) GetLabelSpecs(ctx context.Context) ([]*fleet.LabelSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetLabelSpecs(ctx)
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Spec
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecResponse struct {
	Spec *fleet.LabelSpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getLabelSpecResponse) error() error { return r.Err }

func getLabelSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	return svc.ds.GetLabelSpec(ctx, name)
}

func (svc *Service) BatchValidateLabels(ctx context.Context, labelNames []string) (map[string]fleet.LabelIdent, error) {
	if authctx, ok := authz_ctx.FromContext(ctx); !ok {
		return nil, fleet.NewAuthRequiredError("batch validate labels: missing authorization context")
	} else if !authctx.Checked() {
		return nil, fleet.NewAuthRequiredError("batch validate labels: method requires previous authorization")
	}

	if len(labelNames) == 0 {
		return nil, nil
	}

	uniqueNames := server.RemoveDuplicatesFromSlice(labelNames)

	labels, err := svc.ds.LabelIDsByName(ctx, uniqueNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	if len(labels) != len(uniqueNames) {
		return nil, &fleet.BadRequestError{
			Message:     "some or all the labels provided don't exist",
			InternalErr: fmt.Errorf("names provided: %v", labelNames),
		}
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
