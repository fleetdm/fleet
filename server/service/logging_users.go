package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (mw loggingMiddleware) ChangeUserAdmin(ctx context.Context, id uint, isAdmin bool) (*kolide.User, error) {
	var (
		loggedInUser = "unauthenticated"
		userName     = "none"
		err          error
		user         *kolide.User
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ChangeUserAdmin",
			"user", userName,
			"changed_by", loggedInUser,
			"admin", isAdmin,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.ChangeUserAdmin(ctx, id, isAdmin)
	if user != nil {
		userName = user.Username
	}
	return user, err
}

func (mw loggingMiddleware) CreateUser(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	var (
		user         *kolide.User
		err          error
		username     = "none"
		loggedInUser = "unauthenticated"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "CreateUser",
			"user", username,
			"created_by", loggedInUser,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.CreateUser(ctx, p)
	if user != nil {
		username = user.Username
	}
	return user, err
}

func (mw loggingMiddleware) ListUsers(ctx context.Context, opt kolide.ListOptions) ([]*kolide.User, error) {
	var (
		users    []*kolide.User
		err      error
		username = "none"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		username = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ListUsers",
			"user", username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	users, err = mw.Service.ListUsers(ctx, opt)
	return users, err
}

func (mw loggingMiddleware) RequirePasswordReset(ctx context.Context, uid uint, require bool) (*kolide.User, error) {
	var (
		user     *kolide.User
		err      error
		username = "none"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		username = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "RequirePasswordReset",
			"user", username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.RequirePasswordReset(ctx, uid, require)
	return user, err

}

func (mw loggingMiddleware) CreateUserWithInvite(ctx context.Context, p kolide.UserPayload) (*kolide.User, error) {
	var (
		user         *kolide.User
		err          error
		username     = "none"
		loggedInUser = "unauthenticated"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "CreateUserWithInvite",
			"user", username,
			"created_by", loggedInUser,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.CreateUserWithInvite(ctx, p)

	if user != nil {
		username = user.Username
	}
	return user, err
}

func (mw loggingMiddleware) ModifyUser(ctx context.Context, userID uint, p kolide.UserPayload) (*kolide.User, error) {
	var (
		user     *kolide.User
		err      error
		username = "none"
	)

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errNoContext
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyUser",
			"user", username,
			"modified_by", vc.Username(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.ModifyUser(ctx, userID, p)

	if user != nil {
		username = user.Username
	}

	return user, err
}

func (mw loggingMiddleware) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	var (
		requestedBy = "unauthenticated"
		err         error
	)
	vc, ok := viewer.FromContext(ctx)
	if ok {
		requestedBy = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ChangePassword",
			"err", err,
			"requested_by", requestedBy,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.ChangePassword(ctx, oldPass, newPass)
	return err
}

func (mw loggingMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	var err error

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ResetPassword",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.ResetPassword(ctx, token, password)
	return err
}

func (mw loggingMiddleware) RequestPasswordReset(ctx context.Context, email string) error {
	var (
		requestedBy = "unauthenticated"
		err         error
	)
	vc, ok := viewer.FromContext(ctx)
	if ok {
		requestedBy = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "RequestPasswordReset",
			"email", email,
			"err", err,
			"requested_by", requestedBy,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.RequestPasswordReset(ctx, email)
	return err
}

func (mw loggingMiddleware) PerformRequiredPasswordReset(ctx context.Context, password string) (*kolide.User, error) {
	var (
		resetBy = "unauthenticated"
		err     error
	)
	vc, ok := viewer.FromContext(ctx)
	if ok {
		resetBy = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "PerformRequiredPasswordReset",
			"err", err,
			"reset_by", resetBy,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err := mw.Service.PerformRequiredPasswordReset(ctx, password)
	return user, err
}
