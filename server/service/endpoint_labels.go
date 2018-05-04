package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

type getLabelRequest struct {
	ID uint
}

type labelResponse struct {
	kolide.Label
	DisplayText     string `json:"display_text"`
	Count           uint   `json:"count"`
	Online          uint   `json:"online"`
	Offline         uint   `json:"offline"`
	MissingInAction uint   `json:"missing_in_action"`
	HostIDs         []uint `json:"host_ids"`
}

type getLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r getLabelResponse) error() error { return r.Err }

func labelResponseForLabel(ctx context.Context, svc kolide.Service, label *kolide.Label) (*labelResponse, error) {
	metrics, err := svc.CountHostsInTargets(ctx, nil, []uint{label.ID})
	if err != nil {
		return nil, err
	}
	hosts, err := svc.HostIDsForLabel(label.ID)
	if err != nil {
		return nil, err
	}
	return &labelResponse{
		*label,
		label.Name,
		metrics.TotalHosts,
		metrics.OnlineHosts,
		metrics.OfflineHosts,
		metrics.MissingInActionHosts,
		hosts,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Label
////////////////////////////////////////////////////////////////////////////////

func makeGetLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getLabelRequest)
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
}

////////////////////////////////////////////////////////////////////////////////
// List Labels
////////////////////////////////////////////////////////////////////////////////

type listLabelsRequest struct {
	ListOptions kolide.ListOptions
}

type listLabelsResponse struct {
	Labels []labelResponse `json:"labels"`
	Err    error           `json:"error,omitempty"`
}

func (r listLabelsResponse) error() error { return r.Err }

func makeListLabelsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listLabelsRequest)
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
}

////////////////////////////////////////////////////////////////////////////////
// Delete Label
////////////////////////////////////////////////////////////////////////////////

type deleteLabelRequest struct {
	ID uint
}

type deleteLabelResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteLabelResponse) error() error { return r.Err }

func makeDeleteLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteLabelRequest)
		err := svc.DeleteLabel(ctx, req.ID)
		if err != nil {
			return deleteLabelResponse{Err: err}, nil
		}
		return deleteLabelResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply Label Specs
////////////////////////////////////////////////////////////////////////////////

type applyLabelSpecsRequest struct {
	Specs []*kolide.LabelSpec `json:"specs"`
}

type applyLabelSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyLabelSpecsResponse) error() error { return r.Err }

func makeApplyLabelSpecsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyLabelSpecsRequest)
		err := svc.ApplyLabelSpecs(ctx, req.Specs)
		if err != nil {
			return applyLabelSpecsResponse{Err: err}, nil
		}
		return applyLabelSpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Specs
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecsResponse struct {
	Specs []*kolide.LabelSpec `json:"specs"`
	Err   error               `json:"error,omitempty"`
}

func (r getLabelSpecsResponse) error() error { return r.Err }

func makeGetLabelSpecsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetLabelSpecs(ctx)
		if err != nil {
			return getLabelSpecsResponse{Err: err}, nil
		}
		return getLabelSpecsResponse{Specs: specs}, nil
	}
}
