package schedule

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

func SetWebhooksConfigCheck(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) func(start time.Time, prevInterval time.Duration) (*time.Duration, error) {
	return func(time.Time, time.Duration) (*time.Duration, error) {
		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			level.Error(logger).Log("config", "couldn't read app config", "err", err)
			return nil, err
		}
		newInterval := appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour)

		return &newInterval, nil
	}
}

func DoWebhooks(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	failingPoliciesSet fleet.FailingPolicySet,
) (interface{}, error) {
	stats := make(map[string]string)

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return nil, err
	}

	maybeTriggerHostStatus(ctx, ds, logger, appConfig)
	maybeTriggerGlobalFailingPoliciesWebhook(ctx, ds, logger, appConfig, failingPoliciesSet)
	level.Debug(logger).Log("webhooks", "done")

	return stats, nil
}

func maybeTriggerHostStatus(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
) {
	level.Debug(logger).Log("webhook_host_status", "maybe trigger webhook...")

	if err := webhooks.TriggerHostStatusWebhook(
		ctx, ds, kitlog.With(logger, "webhook", "host_status"), appConfig,
	); err != nil {
		level.Error(logger).Log("err", "triggering host status webhook", "details", err)
		sentry.CaptureException(err)
	}
}

func maybeTriggerGlobalFailingPoliciesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
	failingPoliciesSet fleet.FailingPolicySet,
) {
	level.Debug(logger).Log("webhook_failing_policies", "maybe trigger webhook...")

	if err := webhooks.TriggerGlobalFailingPoliciesWebhook(
		ctx, ds, kitlog.With(logger, "webhook", "failing_policies"), appConfig, failingPoliciesSet, time.Now(),
	); err != nil {
		level.Error(logger).Log("err", "triggering failing policies webhook", "details", err)
		sentry.CaptureException(err)
	}
}
