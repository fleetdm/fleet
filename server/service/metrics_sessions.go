package service

import (
	"fmt"
	"time"

	"github.com/kolide/kolide/server/kolide"
	"golang.org/x/net/context"
)

func (mw metricsMiddleware) Login(ctx context.Context, username string, password string) (*kolide.User, string, error) {
	var (
		user  *kolide.User
		token string
		err   error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Login", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	user, token, err = mw.Service.Login(ctx, username, password)
	return user, token, err
}

func (mw metricsMiddleware) Logout(ctx context.Context) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Logout", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.Logout(ctx)
	return err
}

func (mw metricsMiddleware) DestroySession(ctx context.Context) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "DestroySession", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DestroySession(ctx)
	return err
}

func (mw metricsMiddleware) GetInfoAboutSessionsForUser(ctx context.Context, id uint) ([]*kolide.Session, error) {
	var (
		sessions []*kolide.Session
		err      error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "GetInfoAboutSessionsForUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	sessions, err = mw.Service.GetInfoAboutSessionsForUser(ctx, id)
	return sessions, err
}

func (mw metricsMiddleware) DeleteSessionsForUser(ctx context.Context, id uint) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteSessionsForUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteSessionsForUser(ctx, id)
	return err
}

func (mw metricsMiddleware) GetInfoAboutSession(ctx context.Context, id uint) (*kolide.Session, error) {
	var (
		session *kolide.Session
		err     error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "GetInfoAboutSession", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	session, err = mw.Service.GetInfoAboutSession(ctx, id)
	return session, err
}

func (mw metricsMiddleware) GetSessionByKey(ctx context.Context, key string) (*kolide.Session, error) {
	var (
		session *kolide.Session
		err     error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "GetSessionByKey", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	session, err = mw.Service.GetSessionByKey(ctx, key)
	return session, err
}

func (mw metricsMiddleware) DeleteSession(ctx context.Context, id uint) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteSession", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteSession(ctx, id)
	return err
}
