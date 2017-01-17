package service

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw metricsMiddleware) ChangeUserAdmin(ctx context.Context, id uint, isAdmin bool) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "ChangeUserAdmin", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.ChangeUserAdmin(ctx, id, isAdmin)
	return user, err
}

func (mw metricsMiddleware) ChangeUserEnabled(ctx context.Context, id uint, isEnabled bool) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "ChangeUserEnabled", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.ChangeUserEnabled(ctx, id, isEnabled)
	return user, err
}

func (mw metricsMiddleware) NewUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "NewUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.NewUser(ctx, p)
	return user, err
}

func (mw metricsMiddleware) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.ModifyUser(ctx, userID, p)
	return user, err
}

func (mw metricsMiddleware) User(ctx context.Context, id uint) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "User", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.User(ctx, id)
	return user, err
}

func (mw metricsMiddleware) ListUsers(ctx context.Context, opt kolide.ListOptions) ([]*kolide.User, error) {

	var (
		users []*kolide.User
		err   error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Users", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	users, err = mw.Service.ListUsers(ctx, opt)
	return users, err
}

func (mw metricsMiddleware) AuthenticatedUser(ctx context.Context) (*kolide.User, error) {
	var (
		user *kolide.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "AuthenticatedUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.AuthenticatedUser(ctx)
	return user, err
}

func (mw metricsMiddleware) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "ChangePassword", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.ChangePassword(ctx, oldPass, newPass)
	return err
}

func (mw metricsMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "ResetPassword", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.ResetPassword(ctx, token, password)
	return err
}

func (mw metricsMiddleware) RequestPasswordReset(ctx context.Context, email string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "RequestPasswordReset", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.RequestPasswordReset(ctx, email)
	return err
}
