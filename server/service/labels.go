package service

import (
	"context"

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

func createLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createLabelRequest)

	label, err := svc.NewLabel(ctx, req.LabelPayload)
	if err != nil {
		return createLabelResponse{Err: err}, nil
	}

	labelResp, err := labelResponseForLabel(ctx, svc, label)
	if err != nil {
		return createLabelResponse{Err: err}, nil
	}

	return createLabelResponse{Label: *labelResp}, nil
}

func (svc *Service) NewLabel(ctx context.Context, p fleet.LabelPayload) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	label := &fleet.Label{}

	if p.Name == nil {
		return nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	label.Name = *p.Name

	if p.Query == nil {
		return nil, fleet.NewInvalidArgumentError("query", "missing required argument")
	}
	label.Query = *p.Query

	if p.Platform != nil {
		label.Platform = *p.Platform
	}

	if p.Description != nil {
		label.Description = *p.Description
	}

	label, err := svc.ds.NewLabel(ctx, label)
	if err != nil {
		return nil, err
	}
	return label, nil
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

func modifyLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyLabelRequest)
	label, err := svc.ModifyLabel(ctx, req.ID, req.ModifyLabelPayload)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	labelResp, err := labelResponseForLabel(ctx, svc, label)
	if err != nil {
		return modifyLabelResponse{Err: err}, nil
	}

	return modifyLabelResponse{Label: *labelResp}, err
}

func (svc *Service) ModifyLabel(ctx context.Context, id uint, payload fleet.ModifyLabelPayload) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	label, err := svc.ds.Label(ctx, id)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		label.Name = *payload.Name
	}
	if payload.Description != nil {
		label.Description = *payload.Description
	}
	return svc.ds.SaveLabel(ctx, label)
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
	HostIDs     []uint `json:"host_ids"`
}

type getLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r getLabelResponse) error() error { return r.Err }

func getLabelEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getLabelRequest)
	label, err := svc.GetLabel(ctx, req.ID)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	resp, err := labelResponseForLabel(ctx, svc, label)
	if err != nil {
		return getLabelResponse{Err: err}, nil
	}
	return getLabelResponse{Label: *resp}, nil
}

func (svc *Service) GetLabel(ctx context.Context, id uint) (*fleet.Label, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Label{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Label(ctx, id)
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

func listLabelsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listLabelsRequest)
	labels, err := svc.ListLabels(ctx, req.ListOptions)
	if err != nil {
		return listLabelsResponse{Err: err}, nil
	}

	resp := listLabelsResponse{}
	for _, label := range labels {
		labelResp, err := labelResponseForLabel(ctx, svc, label)
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

	return svc.ds.ListLabels(ctx, filter, opt)
}

func labelResponseForLabel(ctx context.Context, svc fleet.Service, label *fleet.Label) (*labelResponse, error) {
	return &labelResponse{
		Label:       *label,
		DisplayText: label.Name,
		Count:       label.HostCount,
	}, nil
}
