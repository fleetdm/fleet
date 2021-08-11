package service

import (
	"context"
	"encoding/base64"
	"html/template"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/pkg/errors"
)

func (svc Service) InviteNewUser(ctx context.Context, payload fleet.InvitePayload) (*fleet.Invite, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Invite{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// verify that the user with the given email does not already exist
	_, err := svc.ds.UserByEmail(*payload.Email)
	if err == nil {
		return nil, fleet.NewInvalidArgumentError("email", "a user with this account already exists")
	}
	if _, ok := err.(fleet.NotFoundError); !ok {
		return nil, err
	}

	// find the user who created the invite
	v, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("missing viewer context for create invite")
	}
	inviter := v.User

	random, err := server.GenerateRandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))

	invite := &fleet.Invite{
		Email:      *payload.Email,
		InvitedBy:  inviter.ID,
		Token:      token,
		GlobalRole: payload.GlobalRole,
		Teams:      payload.Teams,
	}
	if payload.Position != nil {
		invite.Position = *payload.Position
	}
	if payload.Name != nil {
		invite.Name = *payload.Name
	}
	if payload.SSOEnabled != nil {
		invite.SSOEnabled = *payload.SSOEnabled
	}

	invite, err = svc.ds.NewInvite(invite)
	if err != nil {
		return nil, err
	}

	config, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	invitedBy := inviter.Name
	if invitedBy == "" {
		invitedBy = inviter.Email
	}
	inviteEmail := fleet.Email{
		Subject: "You are Invited to Fleet",
		To:      []string{invite.Email},
		Config:  config,
		Mailer: &mail.InviteMailer{
			Invite:    invite,
			BaseURL:   template.URL(config.ServerURL + svc.config.Server.URLPrefix),
			AssetURL:  getAssetURL(),
			OrgName:   config.OrgName,
			InvitedBy: invitedBy,
		},
	}

	err = svc.mailService.SendEmail(inviteEmail)
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (svc *Service) ListInvites(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Invite, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Invite{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.ListInvites(opt)
}

func (svc *Service) VerifyInvite(ctx context.Context, token string) (*fleet.Invite, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	logging.WithExtras(ctx, "token", token)

	invite, err := svc.ds.InviteByToken(token)
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

func (svc *Service) DeleteInvite(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Invite{}, fleet.ActionWrite); err != nil {
		return err
	}
	return svc.ds.DeleteInvite(id)
}
