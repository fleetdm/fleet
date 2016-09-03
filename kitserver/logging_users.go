package kitserver

import (
	"time"

	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (mw loggingMiddleware) NewUser(ctx context.Context, p kolide.UserPayload) (user *kolide.User, err error) {
	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var username = "none"

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "NewUser",
			"user", username,
			"created_by", vc.user.Username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.NewUser(ctx, p)

	if user != nil {
		username = user.Username
	}
	return
}

func (mw loggingMiddleware) User(ctx context.Context, id uint) (user *kolide.User, err error) {
	var username = "none"

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "User",
			"user", username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, err = mw.Service.User(ctx, id)

	if user != nil {
		username = user.Username
	}
	return
}

func (mw loggingMiddleware) ChangePassword(ctx context.Context, userID uint, old, new string) (err error) {

	vc, err := viewerContextFromContext(ctx)
	if err != nil {
		return err
	}

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ChangePassword",
			"user_id", userID,
			"modified_by", vc.user.Username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.ChangePassword(ctx, userID, old, new)
	return
}
