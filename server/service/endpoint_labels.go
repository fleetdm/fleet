package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide/server/kolide"
	"golang.org/x/net/context"
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
// Create Label
////////////////////////////////////////////////////////////////////////////////

type createLabelRequest struct {
	payload kolide.LabelPayload
}

type createLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r createLabelResponse) error() error { return r.Err }

func makeCreateLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createLabelRequest)

		label, err := svc.NewLabel(ctx, req.payload)
		if err != nil {
			return createLabelResponse{Err: err}, nil
		}

		labelResp, err := labelResponseForLabel(ctx, svc, label)
		if err != nil {
			return createLabelResponse{Err: err}, nil
		}

		return createLabelResponse{Label: *labelResp}, nil
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
// Modify Label
////////////////////////////////////////////////////////////////////////////////

type modifyLabelRequest struct {
	ID      uint
	payload kolide.ModifyLabelPayload
}

type modifyLabelResponse struct {
	Label labelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r modifyLabelResponse) error() error { return r.Err }

func makeModifyLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyLabelRequest)
		label, err := svc.ModifyLabel(ctx, req.ID, req.payload)
		if err != nil {
			return modifyLabelResponse{Err: err}, nil
		}

		labelResp, err := labelResponseForLabel(ctx, svc, label)
		if err != nil {
			return modifyLabelResponse{Err: err}, nil
		}

		return modifyLabelResponse{Label: *labelResp}, err
	}
}
