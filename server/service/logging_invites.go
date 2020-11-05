package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
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
		_ = mw.loggerInfo(err).Log(
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
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteInvite",
			"deleted_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeleteInvite(ctx, id)
	return err
}

func (mw loggingMiddleware) ListInvites(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Invite, error) {
	var (
		invites []*kolide.Invite
		err     error
	)
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "Invites",
			"called_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	invites, err = mw.Service.ListInvites(ctx, opt)
	return invites, err
}

func (mw loggingMiddleware) VerifyInvite(ctx context.Context, token string) (*kolide.Invite, error) {
	var (
		err    error
		invite *kolide.Invite
	)
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "VerifyInvite",
			"token", token,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	invite, err = mw.Service.VerifyInvite(ctx, token)
	return invite, err
}
