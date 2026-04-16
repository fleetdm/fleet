package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
)

const (
	notificationDeliveriesBatch = 50
	// Pick a periodicity that feels snappy for ops without hammering the DB.
	// Each tick is one SELECT + a handful of HTTP POSTs against webhooks.
	notificationDeliveriesPeriodicity = 1 * time.Minute
)

// slackSender is an injection point so the cron can be exercised in tests
// without hitting real Slack webhooks. The default uses PostSlackWebhook in
// the service package (wired at startup via NewNotificationDeliveriesSchedule).
//
// serverBaseURL is threaded through so the deliverer can absolutize any
// relative CTA URLs on the notification — Slack rejects Block Kit buttons
// whose url field isn't absolute.
type slackSender func(ctx context.Context, url string, n *fleet.Notification, serverBaseURL string) error

// NewNotificationDeliveriesSchedule returns the cron that drains pending
// notification_deliveries rows — today just Slack, but the dispatcher is
// channel-keyed so email/slack/future channels all share the same plumbing.
func NewNotificationDeliveriesSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	sendSlack slackSender,
	logger *slog.Logger,
) (*schedule.Schedule, error) {
	name := string(fleet.CronNotificationDeliveries)
	logger = logger.With("cron", name)
	s := schedule.New(
		ctx, name, instanceID, notificationDeliveriesPeriodicity, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob(
			"deliver_slack",
			func(ctx context.Context) error {
				return processPendingSlackDeliveries(ctx, ds, sendSlack, logger)
			},
		),
	)
	return s, nil
}

// processPendingSlackDeliveries claims a batch of pending slack deliveries
// and dispatches each to its webhook URL. Failures are recorded on the
// delivery row rather than returned, so one bad webhook doesn't poison the
// batch — a single totally-unreachable integration shouldn't silence the
// rest of the fleet's deliveries.
func processPendingSlackDeliveries(
	ctx context.Context, ds fleet.Datastore, sendSlack slackSender, logger *slog.Logger,
) error {
	deliveries, notifs, err := ds.ClaimPendingDeliveries(ctx, fleet.NotificationChannelSlack, notificationDeliveriesBatch)
	if err != nil {
		return fmt.Errorf("claim pending deliveries: %w", err)
	}
	if len(deliveries) == 0 {
		return nil
	}

	// Load the server URL once per batch so the deliverer can absolutize
	// relative CTA paths on the notification. A missing server URL isn't
	// fatal — the deliverer just drops the CTA button.
	var serverBaseURL string
	if cfg, err := ds.AppConfig(ctx); err != nil {
		logger.WarnContext(ctx, "load app config for slack server_url", "err", err)
	} else {
		serverBaseURL = cfg.ServerSettings.ServerURL
	}

	for _, d := range deliveries {
		notif, ok := notifs[d.NotificationID]
		if !ok {
			// The notification got deleted between enqueue and dispatch
			// (cascade or manual cleanup). Mark failed with a stable reason
			// so it doesn't re-queue; there's nothing to send.
			if err := ds.MarkDeliveryResult(ctx, d.ID, fleet.NotificationDeliveryStatusFailed,
				"source notification no longer exists"); err != nil {
				logger.ErrorContext(ctx, "mark delivery failed", "delivery_id", d.ID, "err", err)
			}
			continue
		}

		if err := sendSlack(ctx, d.Target, notif, serverBaseURL); err != nil {
			logger.WarnContext(ctx, "slack delivery failed",
				"delivery_id", d.ID, "notification_id", d.NotificationID, "err", err)
			if mErr := ds.MarkDeliveryResult(ctx, d.ID, fleet.NotificationDeliveryStatusFailed, err.Error()); mErr != nil {
				logger.ErrorContext(ctx, "mark delivery failed", "delivery_id", d.ID, "err", mErr)
			}
			continue
		}
		if err := ds.MarkDeliveryResult(ctx, d.ID, fleet.NotificationDeliveryStatusSent, ""); err != nil {
			logger.ErrorContext(ctx, "mark delivery sent", "delivery_id", d.ID, "err", err)
		}
	}
	return nil
}
