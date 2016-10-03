package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Get Label
////////////////////////////////////////////////////////////////////////////////

type getLabelRequest struct {
	ID uint
}

type getLabelResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	QueryID uint   `json:"query_id"`
	Err     error  `json:"error,omitempty"`
}

func (r getLabelResponse) error() error { return r.Err }

func makeGetLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getLabelRequest)
		label, err := svc.GetLabel(ctx, req.ID)
		if err != nil {
			return getLabelResponse{Err: err}, nil
		}
		return getLabelResponse{
			ID:      label.ID,
			Name:    label.Name,
			QueryID: label.QueryID,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get All Labels
////////////////////////////////////////////////////////////////////////////////

type getAllLabelsResponse struct {
	Labels []getLabelResponse `json:"labels"`
	Err    error              `json:"error,omitempty"`
}

func (r getAllLabelsResponse) error() error { return r.Err }

func makeGetAllLabelsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		labels, err := svc.GetAllLabels(ctx)
		if err != nil {
			return getAllLabelsResponse{Err: err}, nil
		}
		var resp getAllLabelsResponse
		for _, label := range labels {
			resp.Labels = append(resp.Labels, getLabelResponse{
				ID:      label.ID,
				Name:    label.Name,
				QueryID: label.QueryID,
			})
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
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	QueryID uint   `json:"query_id"`
	Err     error  `json:"error,omitempty"`
}

func (r createLabelResponse) error() error { return r.Err }

func makeCreateLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createLabelRequest)
		label, err := svc.NewLabel(ctx, req.payload)
		if err != nil {
			return createLabelResponse{Err: err}, nil
		}
		return createLabelResponse{
			ID:      label.ID,
			Name:    label.Name,
			QueryID: label.QueryID,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Label
////////////////////////////////////////////////////////////////////////////////

type modifyLabelRequest struct {
	ID      uint
	payload kolide.LabelPayload
}

type modifyLabelResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	QueryID uint   `json:"query_id"`
	Err     error  `json:"error,omitempty"`
}

func (r modifyLabelResponse) error() error { return r.Err }

func makeModifyLabelEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyLabelRequest)
		label, err := svc.ModifyLabel(ctx, req.ID, req.payload)
		if err != nil {
			return modifyLabelResponse{Err: err}, nil
		}
		return modifyLabelResponse{
			ID:      label.ID,
			Name:    label.Name,
			QueryID: label.QueryID,
		}, nil
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
