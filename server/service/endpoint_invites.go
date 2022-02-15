package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type verifyInviteRequest struct {
	Token string
}

type verifyInviteResponse struct {
	Invite *fleet.Invite `json:"invite"`
	Err    error         `json:"error,omitempty"`
}

func (r verifyInviteResponse) error() error { return r.Err }

func makeVerifyInviteEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(verifyInviteRequest)
		invite, err := svc.VerifyInvite(ctx, req.Token)
		if err != nil {
			return verifyInviteResponse{Err: err}, nil
		}
		return verifyInviteResponse{Invite: invite}, nil
	}
}
