package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) InviteNewUser(ctx context.Context, payload fleet.InvitePayload) (*fleet.Invite, error) {
	var (
		invite *fleet.Invite
		err    error
	)

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "InviteNewUser",
			"created_by", vc.Email(),
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
		return fleet.ErrNoContext
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteInvite",
			"deleted_by", vc.Email(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeleteInvite(ctx, id)
	return err
}

func (mw loggingMiddleware) ListInvites(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Invite, error) {
	var (
		invites []*fleet.Invite
		err     error
	)
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "Invites",
			"called_by", vc.Email(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	invites, err = mw.Service.ListInvites(ctx, opt)
	return invites, err
}

func (mw loggingMiddleware) VerifyInvite(ctx context.Context, token string) (*fleet.Invite, error) {
	var (
		err    error
		invite *fleet.Invite
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
