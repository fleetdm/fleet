package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw metricsMiddleware) ModifyLabel(ctx context.Context, id uint, p kolide.ModifyLabelPayload) (*kolide.Label, error) {
	var (
		lic *kolide.Label
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyLabel", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	lic, err = mw.Service.ModifyLabel(ctx, id, p)
	return lic, err

}
