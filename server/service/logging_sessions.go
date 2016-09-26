package service

import (
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw loggingMiddleware) Login(ctx context.Context, username, password string) (user *kolide.User, token string, err error) {

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "Login",
			"user", username,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	user, token, err = mw.Service.Login(ctx, username, password)
	return
}

func (mw loggingMiddleware) Logout(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "Logout",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.Logout(ctx)
	return
}
