package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw metricsMiddleware) InviteNewUser(ctx context.Context, payload kolide.InvitePayload) (*kolide.Invite, error) {
	var (
		invite *kolide.Invite
		err    error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "InviteNewUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invite, err = mw.Service.InviteNewUser(ctx, payload)
	return invite, err
}

func (mw metricsMiddleware) DeleteInvite(ctx context.Context, id uint) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteInvite", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteInvite(ctx, id)
	return err
}

func (mw metricsMiddleware) ListInvites(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Invite, error) {
	var (
		invites []*kolide.Invite
		err     error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Invites", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invites, err = mw.Service.ListInvites(ctx, opt)
	return invites, err
}

func (mw metricsMiddleware) VerifyInvite(ctx context.Context, token string) (*kolide.Invite, error) {
	var (
		err    error
		invite *kolide.Invite
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "VerifyInvite", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invite, err = mw.Service.VerifyInvite(ctx, token)
	return invite, err
}
