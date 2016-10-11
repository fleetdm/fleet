package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

type createInviteRequest struct {
	payload kolide.InvitePayload
}

type createInviteResponse struct {
	Invite *kolide.Invite `json:"invite,omitempty"`
	Err    error          `json:"error,omitempty"`
}

func (r createInviteResponse) error() error { return r.Err }

func makeCreateInviteEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createInviteRequest)
		invite, err := svc.InviteNewUser(ctx, req.payload)
		if err != nil {
			return createInviteResponse{Err: err}, nil
		}
		return createInviteResponse{invite, nil}, nil
	}
}

type listInvitesResponse struct {
	Invites []kolide.Invite `json:"invites"`
	Err     error           `json:"error,omitempty"`
}

func (r listInvitesResponse) error() error { return r.Err }

func makeListInvitesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		invites, err := svc.Invites(ctx)
		if err != nil {
			return listInvitesResponse{Err: err}, nil
		}

		resp := listInvitesResponse{Invites: []kolide.Invite{}}
		for _, invite := range invites {
			resp.Invites = append(resp.Invites, *invite)
		}
		return resp, nil
	}
}

type deleteInviteRequest struct {
	ID uint
}

type deleteInviteResponse struct {
	Err error `json:"error,omitempty"`
}

func makeDeleteInviteEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteInviteRequest)
		err := svc.DeleteInvite(ctx, req.ID)
		if err != nil {
			return deleteInviteResponse{Err: err}, nil
		}
		return deleteInviteResponse{}, nil
	}
}
