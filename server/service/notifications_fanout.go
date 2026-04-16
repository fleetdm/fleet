package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// fanoutSlackForNotification enqueues one pending notification_deliveries row
// per matching slack_notifications route. Called after every successful
// UpsertNotification in the service layer.
//
// The unique index on (notification_id, channel, target) makes this
// idempotent: producers that re-upsert the same dedupe key on a cron tick
// won't accumulate extra delivery rows. We still call it every time because
// a new notification (different dedupe key) needs its own deliveries.
func fanoutSlackForNotification(ctx context.Context, ds fleet.Datastore, notif *fleet.Notification) error {
	if notif == nil {
		return nil
	}
	cfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config for slack fanout")
	}
	category := fleet.CategoryForType(notif.Type)
	for _, route := range cfg.Integrations.SlackNotifications.Routes {
		// "all" is a wildcard route — it matches every category so admins can
		// mirror every Fleet notification to a firehose channel with a single
		// row instead of one row per category.
		if route.Category != fleet.NotificationCategoryAll && route.Category != category {
			continue
		}
		if err := ds.EnqueueNotificationDelivery(ctx, notif.ID, fleet.NotificationChannelSlack, route.WebhookURL); err != nil {
			return ctxerr.Wrap(ctx, err, "enqueue slack delivery")
		}
	}
	return nil
}

// upsertNotification is the one canonical write path for notifications in the
// service layer: it upserts the row and fans out any admin-configured Slack
// routes. Use it instead of calling svc.ds.UpsertNotification directly.
//
// Fanout errors are logged but not returned — a transient app_config read or
// delivery-row insert failure shouldn't reject an otherwise-valid producer
// upsert (the next cron tick will retry fanout via the INSERT IGNORE).
func (svc *Service) upsertNotification(ctx context.Context, u fleet.NotificationUpsert) (*fleet.Notification, error) {
	n, err := svc.ds.UpsertNotification(ctx, u)
	if err != nil {
		return nil, err
	}
	if err := fanoutSlackForNotification(ctx, svc.ds, n); err != nil {
		svc.logger.WarnContext(ctx, "slack notification fanout failed",
			"notification_id", n.ID, "type", n.Type, "err", err)
	}
	return n, nil
}

// PostSlackWebhook POSTs a Block Kit payload for the given notification to a
// Slack incoming-webhook URL. Returns the error verbatim so the worker can
// persist a meaningful failure reason.
//
// serverBaseURL is used to absolutize any relative CTA URL on the
// notification — Slack's Block Kit button.url field rejects relative paths
// with invalid_blocks. Pass "" to disable CTA buttons entirely for
// notifications that only carry a relative path.
//
// Exported so the cron worker in server/cron can inject this as the Slack
// sender without creating a direct dependency between packages.
func PostSlackWebhook(ctx context.Context, url string, notif *fleet.Notification, serverBaseURL string) error {
	body, err := json.Marshal(slackPayload(notif, serverBaseURL))
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("slack webhook POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	// Slack returns plain-text errors on the webhook endpoint; surface a
	// truncated version so the delivery row captures context.
	preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("slack webhook returned %d: %s",
		resp.StatusCode, strings.TrimSpace(string(preview)))
}

// slackPayload renders a notification to Slack's Block Kit format. The layout
// is kept deliberately compact so it fits cleanly in a channel without being
// noisy: one header (severity-prefixed), one body section, and an optional
// action button pointing at the notification's CTA.
//
// Slack enforces a hard 150-char limit on plain_text.text in header blocks
// (invalid_blocks otherwise) and requires button.url to be an absolute
// http(s) URL. Both are handled here so producers can keep emitting
// Fleet-internal relative paths like "/policies" without Slack failures.
func slackPayload(n *fleet.Notification, serverBaseURL string) map[string]interface{} {
	prefix := severityPrefix(n.Severity)
	headerText := truncateForSlackHeader(fmt.Sprintf("%s %s", prefix, n.Title))
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type":  "plain_text",
				"text":  headerText,
				"emoji": true,
			},
		},
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": n.Body,
			},
		},
	}
	if btnURL := resolveCTAURL(n.CTAURL, serverBaseURL); btnURL != "" {
		label := "View"
		if n.CTALabel != nil && *n.CTALabel != "" {
			label = *n.CTALabel
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "actions",
			"elements": []map[string]interface{}{
				{
					"type": "button",
					"text": map[string]interface{}{
						"type":  "plain_text",
						"text":  label,
						"emoji": false,
					},
					"url": btnURL,
				},
			},
		})
	}
	return map[string]interface{}{
		"text":   headerText,
		"blocks": blocks,
	}
}

// resolveCTAURL returns a Slack-safe absolute URL for the notification's
// call-to-action, or "" if we can't build one. Absolute URLs are passed
// through unchanged; relative paths are joined to serverBaseURL. If the
// CTA is relative and no serverBaseURL is configured, we drop the button
// rather than send Slack a value it will reject with invalid_blocks.
func resolveCTAURL(cta *string, serverBaseURL string) string {
	if cta == nil || *cta == "" {
		return ""
	}
	raw := strings.TrimSpace(*cta)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	base := strings.TrimRight(serverBaseURL, "/")
	if base == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return base + raw
}

// truncateForSlackHeader enforces Slack's 150-char limit on header
// plain_text. Ellipsize mid-word if necessary rather than failing the send.
func truncateForSlackHeader(s string) string {
	const max = 150
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// severityPrefix picks an emoji that makes severity scannable in a busy
// channel. Slack already renders these inline — no extra setup required.
func severityPrefix(s fleet.NotificationSeverity) string {
	switch s {
	case fleet.NotificationSeverityError:
		return ":rotating_light:"
	case fleet.NotificationSeverityWarning:
		return ":warning:"
	default:
		return ":information_source:"
	}
}
