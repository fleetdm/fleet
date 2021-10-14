package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get Pack
////////////////////////////////////////////////////////////////////////////////

type getPackRequest struct {
	ID uint `url:"id"`
}

type getPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r getPackResponse) error() error { return r.Err }

func getPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getPackRequest)
	pack, err := svc.GetPack(ctx, req.ID)
	if err != nil {
		return getPackResponse{Err: err}, nil
	}

	resp, err := packResponseForPack(ctx, svc, *pack)
	if err != nil {
		return getPackResponse{Err: err}, nil
	}

	return getPackResponse{
		Pack: *resp,
	}, nil
}

func (svc *Service) GetPack(ctx context.Context, id uint) (*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Pack(ctx, id)
}
