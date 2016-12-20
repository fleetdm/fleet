package service

import (
	"encoding/base64"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

type inviteMailer struct{}

func (m *inviteMailer) Message() ([]byte, error) {
	return []byte("test message"), nil
}

func (svc service) InviteNewUser(ctx context.Context, payload kolide.InvitePayload) (*kolide.Invite, error) {
	// verify that the user with the given email does not already exist
	_, err := svc.ds.UserByEmail(*payload.Email)
	if err == nil {
		return nil, newInvalidArgumentError("email", "a user with this account already exists")
	}

	if _, ok := err.(kolide.NotFoundError); !ok {
		return nil, err
	}

	// find the user who created the invite
	inviter, err := svc.User(ctx, *payload.InvitedBy)
	if err != nil {
		return nil, err
	}

	random, err := kolide.RandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))

	invite := &kolide.Invite{
		Email:     *payload.Email,
		Admin:     *payload.Admin,
		InvitedBy: inviter.ID,
		Token:     token,
	}
	if payload.Position != nil {
		invite.Position = *payload.Position
	}
	if payload.Name != nil {
		invite.Name = *payload.Name
	}

	invite, err = svc.ds.NewInvite(invite)
	if err != nil {
		return nil, err
	}

	config, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	inviteEmail := kolide.Email{
		Subject: "You're Invited to Kolide",
		To:      []string{invite.Email},
		Config:  config,
		Mailer:  &inviteMailer{},
	}

	err = svc.mailService.SendEmail(inviteEmail)
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (svc service) ListInvites(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Invite, error) {
	return svc.ds.ListInvites(opt)
}

func (svc service) VerifyInvite(ctx context.Context, email, token string) error {
	invite, err := svc.ds.InviteByEmail(email)
	if err != nil {
		return err
	}

	if invite.Token != token {
		return newInvalidArgumentError("invite_token", "Invite Token does not match Email Address.")
	}

	expiresAt := invite.CreatedAt.Add(svc.config.App.InviteTokenValidityPeriod)
	if svc.clock.Now().After(expiresAt) {
		return newInvalidArgumentError("invite_token", "Invite token has expired.")
	}

	return nil

}

func (svc service) DeleteInvite(ctx context.Context, id uint) error {
	invite, err := svc.ds.Invite(id)
	if err != nil {
		return err
	}
	return svc.ds.DeleteInvite(invite)
}
