package service

import (
	"time"

	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw loggingMiddleware) InviteNewUser(ctx context.Context, payload kolide.InvitePayload) (*kolide.Invite, error) {
	var (
		invite *kolide.Invite
		err    error
	)

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "InviteNewUser",
			"created_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	invite, err = mw.Service.InviteNewUser(ctx, payload)
	return invite, err
}

func (mw loggingMiddleware) DeleteInvite(ctx context.Context, id uint) error {
	var (
		err error
	)
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errNoContext
	}
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "DeleteInvite",
			"deleted_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeleteInvite(ctx, id)
	return err
}

func (mw loggingMiddleware) Invites(ctx context.Context) ([]*kolide.Invite, error) {
	var (
		invites []*kolide.Invite
		err     error
	)
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "Invites",
			"called_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	invites, err = mw.Service.Invites(ctx)
	return invites, err
}

func (mw loggingMiddleware) VerifyInvite(ctx context.Context, email string, token string) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "VerifyInvite",
			"email", email,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.VerifyInvite(ctx, email, token)
	return err
}
