package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (mw loggingMiddleware) GetOptionsSpec(ctx context.Context) (spec *kolide.OptionsSpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetOptionsSpec",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	spec, err = mw.Service.GetOptionsSpec(ctx)
	return spec, err
}

func (mw loggingMiddleware) ApplyOptionsSpec(ctx context.Context, spec *kolide.OptionsSpec) (err error) {
	var (
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ApplyOptionsSpec",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ApplyOptionsSpec(ctx, spec)
	return err
}
