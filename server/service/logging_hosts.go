package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) ListHosts(ctx context.Context, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	var (
		hosts []*fleet.Host
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListHosts",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	hosts, err = mw.Service.ListHosts(ctx, opt)
	return hosts, err
}

func (mw loggingMiddleware) GetHost(ctx context.Context, id uint) (*fleet.HostDetail, error) {
	var (
		host *fleet.HostDetail
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetHost",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	host, err = mw.Service.GetHost(ctx, id)
	return host, err
}

func (mw loggingMiddleware) GetHostSummary(ctx context.Context) (*fleet.HostSummary, error) {
	var (
		summary *fleet.HostSummary
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetHostSummary",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	summary, err = mw.Service.GetHostSummary(ctx)
	return summary, err
}

func (mw loggingMiddleware) DeleteHost(ctx context.Context, id uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteHost",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeleteHost(ctx, id)
	return err
}
