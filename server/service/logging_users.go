package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) CreateUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var (
		user         *fleet.User
		err          error
		email        = "<none>"
		loggedInUser = "unauthenticated"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "CreateUser",
			"user", email,
			"created_by", loggedInUser,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.CreateUser(ctx, p)
	if user != nil {
		email = user.Email
	}
	return user, err
}

func (mw loggingMiddleware) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	var (
		users []*fleet.User
		err   error
		email = "<none>"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		email = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ListUsers",
			"user", email,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	users, err = mw.Service.ListUsers(ctx, opt)
	return users, err
}

func (mw loggingMiddleware) RequirePasswordReset(ctx context.Context, uid uint, require bool) (*fleet.User, error) {
	var (
		user  *fleet.User
		err   error
		email = "<none>"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		email = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "RequirePasswordReset",
			"user", email,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.RequirePasswordReset(ctx, uid, require)
	return user, err

}

func (mw loggingMiddleware) CreateUserFromInvite(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var (
		user         *fleet.User
		err          error
		email        = "<none>"
		loggedInUser = "unauthenticated"
	)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "CreateUserFromInvite",
			"user", email,
			"created_by", loggedInUser,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.CreateUserFromInvite(ctx, p)

	if user != nil {
		email = user.Email
	}
	return user, err
}

func (mw loggingMiddleware) ModifyUser(ctx context.Context, userID uint, p fleet.UserPayload) (*fleet.User, error) {
	var (
		user  *fleet.User
		err   error
		email = "<none>"
	)

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyUser",
			"user", email,
			"modified_by", vc.Email(),
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.ModifyUser(ctx, userID, p)

	if user != nil {
		email = user.Email
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
		requestedBy = vc.Email()
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
		requestedBy = vc.Email()
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

func (mw loggingMiddleware) PerformRequiredPasswordReset(ctx context.Context, password string) (*fleet.User, error) {
	var (
		resetBy = "unauthenticated"
		err     error
	)
	vc, ok := viewer.FromContext(ctx)
	if ok {
		resetBy = vc.Email()
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
