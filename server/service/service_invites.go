package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) VerifyInvite(ctx context.Context, token string) (*fleet.Invite, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	logging.WithExtras(ctx, "token", token)

	invite, err := svc.ds.InviteByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if invite.Token != token {
		return nil, fleet.NewInvalidArgumentError("invite_token", "Invite Token does not match Email Address.")
	}

	expiresAt := invite.CreatedAt.Add(svc.config.App.InviteTokenValidityPeriod)
	if svc.clock.Now().After(expiresAt) {
		return nil, fleet.NewInvalidArgumentError("invite_token", "Invite token has expired.")
	}

	return invite, nil

}
