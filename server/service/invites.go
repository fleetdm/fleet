package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Update invite
////////////////////////////////////////////////////////////////////////////////

type updateInviteRequest struct {
	ID uint `url:"id"`
	fleet.InvitePayload
}

type updateInviteResponse struct {
	Invite *fleet.Invite `json:"invite"`
	Err    error         `json:"error,omitempty"`
}

func (r updateInviteResponse) error() error { return r.Err }

func updateInviteEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*updateInviteRequest)
	invite, err := svc.UpdateInvite(ctx, req.ID, req.InvitePayload)
	if err != nil {
		return updateInviteResponse{Err: err}, nil
	}

	return updateInviteResponse{Invite: invite}, nil
}

func (svc *Service) UpdateInvite(ctx context.Context, id uint, payload fleet.InvitePayload) (*fleet.Invite, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Invite{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := fleet.ValidateRole(payload.GlobalRole.Ptr(), payload.Teams); err != nil {
		return nil, err
	}

	invite, err := svc.ds.Invite(ctx, id)
	if err != nil {
		return nil, err
	}

	if payload.Email != nil {
		invite.Email = *payload.Email
	}
	if payload.Name != nil {
		invite.Name = *payload.Name
	}
	if payload.Position != nil {
		invite.Position = *payload.Position
	}
	if payload.SSOEnabled != nil {
		invite.SSOEnabled = *payload.SSOEnabled
	}
	invite.GlobalRole = payload.GlobalRole
	invite.Teams = payload.Teams

	return svc.ds.UpdateInvite(ctx, id, invite)
}
