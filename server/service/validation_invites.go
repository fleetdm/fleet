package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw validationMiddleware) InviteNewUser(ctx context.Context, payload fleet.InvitePayload) (*fleet.Invite, error) {
	invalid := &fleet.InvalidArgumentError{}
	if payload.Email == nil {
		invalid.Append("email", "missing required argument")
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.InviteNewUser(ctx, payload)
}
