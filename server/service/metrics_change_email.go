package service

import (
	"context"
	"fmt"
	"time"
)

func (mw metricsMiddleware) ChangeUserEmail(ctx context.Context, token string) (string, error) {
	var (
		err      error
		newEmail string
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "CommitEmailChange", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	newEmail, err = mw.Service.ChangeUserEmail(ctx, token)
	return newEmail, err
}
