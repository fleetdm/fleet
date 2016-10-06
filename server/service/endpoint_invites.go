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
	ID        uint   `json:"id"`
	InvitedBy uint   `json:"invited_by"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Admin     bool   `json:"admin"`
	Position  string `json:"position,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r createInviteResponse) error() error { return r.Err }

func makeCreateInviteEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createInviteRequest)
		invite, err := svc.InviteNewUser(ctx, req.payload)
		if err != nil {
			return createInviteResponse{Err: err}, nil
		}
		return createInviteResponse{
			ID:        invite.ID,
			InvitedBy: invite.InvitedBy,
			Email:     invite.Email,
			Name:      invite.Name,
			Position:  invite.Position,
			Admin:     invite.Admin,
		}, nil
	}
}

type inviteResponse struct {
	ID        uint   `json:"id"`
	InvitedBy uint   `json:"invited_by"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Admin     bool   `json:"admin"`
	Position  string `json:"position,omitempty"`
}

type listInvitesResponse struct {
	Invites []inviteResponse `json:"invites"`
	Err     error            `json:"error,omitempty"`
}

func (r listInvitesResponse) error() error { return r.Err }

func makeListInvitesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		invites, err := svc.Invites(ctx)
		if err != nil {
			return listInvitesResponse{Err: err}, nil
		}

		resp := listInvitesResponse{Invites: []inviteResponse{}}
		for _, invite := range invites {
			resp.Invites = append(resp.Invites, inviteResponse{
				ID:        invite.ID,
				InvitedBy: invite.InvitedBy,
				Email:     invite.Email,
				Name:      invite.Name,
				Admin:     invite.Admin,
				Position:  invite.Position,
			})
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
