package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw metricsMiddleware) SSOSettings(ctx context.Context) (settings *fleet.SessionSSOSettings, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "SessionSSOSettings", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	settings, err = mw.Service.SSOSettings(ctx)
	return
}

func (mw metricsMiddleware) InitiateSSO(ctx context.Context, relayValue string) (idpURL string, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "InitiateSSO", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	idpURL, err = mw.Service.InitiateSSO(ctx, relayValue)
	return
}

func (mw metricsMiddleware) CallbackSSO(ctx context.Context, auth fleet.Auth) (sess *fleet.SSOSession, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "CallbackSSO", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	sess, err = getSSOSession(ctx, mw.Service, auth)
	return
}

func (mw metricsMiddleware) Login(ctx context.Context, email string, password string, supportsEmailVerification bool) (*fleet.User, *fleet.Session, error) {
	var (
		user    *fleet.User
		session *fleet.Session
		err     error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Login", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	user, session, err = mw.Service.Login(ctx, email, password, supportsEmailVerification)
	return user, session, err
}

func (mw metricsMiddleware) Logout(ctx context.Context) error {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "Logout", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.Logout(ctx)
	return err
}

func (mw metricsMiddleware) DestroySession(ctx context.Context) error {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "DestroySession", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DestroySession(ctx)
	return err
}

func (mw metricsMiddleware) GetInfoAboutSessionsForUser(ctx context.Context, id uint) ([]*fleet.Session, error) {
	var (
		sessions []*fleet.Session
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
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteSessionsForUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteSessionsForUser(ctx, id)
	return err
}

func (mw metricsMiddleware) GetInfoAboutSession(ctx context.Context, id uint) (*fleet.Session, error) {
	var (
		session *fleet.Session
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

func (mw metricsMiddleware) DeleteSession(ctx context.Context, id uint) error {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteSession", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteSession(ctx, id)
	return err
}
