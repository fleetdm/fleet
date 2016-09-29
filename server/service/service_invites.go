package service

import (
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) InviteNewUser(ctx context.Context, payload kolide.InvitePayload) (*kolide.Invite, error) {
	// verify that the user with the given email does not already exist
	_, err := svc.ds.UserByEmail(*payload.Email)
	if err == nil {
		return nil, newInvalidArgumentError("email", "a user with this account already exists")
	}
	if err != datastore.ErrNotFound {
		return nil, err
	}

	// find the user who created the invite
	inviter, err := svc.User(ctx, *payload.InvitedBy)
	if err != nil {
		return nil, err
	}

	token, err := jwt.New(jwt.SigningMethodHS256).SignedString([]byte(svc.config.App.TokenKey))
	if err != nil {
		return nil, err
	}

	invite := &kolide.Invite{
		Email:     *payload.Email,
		Admin:     *payload.Admin,
		InvitedBy: inviter.ID,
		CreatedAt: svc.clock.Now(),
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

	inviteEmail := kolide.Email{
		From: "no-reply@kolide.co",
		To:   []string{invite.Email},
		Msg:  invite,
	}
	err = svc.mailService.SendEmail(inviteEmail)
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (svc service) Invites(ctx context.Context) ([]*kolide.Invite, error) {
	return svc.ds.Invites()
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
		return errors.New("expired invite token")
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
