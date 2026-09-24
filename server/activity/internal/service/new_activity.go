package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platformhttp "github.com/fleetdm/fleet/v4/server/platform/http"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// webhookPayload is the payload sent to the activities webhook.
type webhookPayload struct {
	Timestamp     time.Time        `json:"timestamp"`
	ActorFullName *string          `json:"actor_full_name"`
	ActorID       *uint            `json:"actor_id"`
	ActorEmail    *string          `json:"actor_email"`
	Type          string           `json:"type"`
	Details       *json.RawMessage `json:"details"`
}

// NewActivity creates a new activity record and fires the webhook if configured.
func (s *Service) NewActivity(ctx context.Context, user *api.User, activity api.ActivityDetails) error {
	detailsBytes, err := json.Marshal(activity)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling activity details")
	}
	// Duplicate JSON keys so that stored activity details include both the
	// old and new field names (e.g. team_id and fleet_id).
	if rules := eu.ExtractAliasRules(activity); len(rules) > 0 {
		detailsBytes = eu.DuplicateJSONKeys(detailsBytes, rules, eu.DuplicateJSONKeysOpts{Compact: true})
	}
	timestamp := time.Now()

	// Fire webhook if enabled
	webhookConfig, err := s.providers.GetActivitiesWebhookConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get activities webhook config")
	}

	if webhookConfig != nil && webhookConfig.Enable {
		s.fireActivityWebhook(ctx, user, activity, detailsBytes, timestamp, webhookConfig.DestinationURL)
	}

	// Fire the per-fleet host activities webhooks if the activity is linked to
	// hosts and any of those hosts' fleets has the webhook enabled.
	if ah, ok := activity.(types.ActivityHosts); ok {
		if hostIDs := ah.HostIDs(); len(hostIDs) > 0 {
			hooks, err := s.providers.GetHostActivitiesWebhooks(ctx, hostIDs)
			if err != nil {
				// transient failure here must not fail the surrounding operation recording the activity.
				s.logger.ErrorContext(ctx, "get host activities webhooks",
					slog.String("activity", activity.ActivityName()), slog.String("err", err.Error()))
				// Discard partial results: an incomplete hook list could
				// deliver to some fleets and silently miss others.
				hooks = nil
			}
			// Per-fleet payloads carry host_ids for activities whose stored
			// details don't already identify their hosts (batch automation
			// activities). The list is injected at fire time only — batches can
			// span thousands of hosts, so storing it would bloat every affected
			// host's feed response and expose the full batch to any user who
			// can read one of its hosts — and each delivery gets only the host
			// IDs belonging to the fleet(s) behind its destination, so one
			// fleet's endpoint never sees another fleet's hosts. The global
			// webhook payload above is intentionally left as stored (matching
			// its pre-existing format).
			for _, hook := range hooks {
				webhookDetails := s.detailsWithHostIDs(ctx, activity.ActivityName(), detailsBytes, hook.HostIDs)
				s.fireActivityWebhook(ctx, user, activity, webhookDetails, timestamp, hook.DestinationURL)
			}
		}
	}

	// Activate the next upcoming activity if requested by the activity type.
	// This is done before storing to avoid holding a DB transaction open during
	// potentially slow operations.
	if aa, ok := activity.(types.ActivityActivator); ok && aa.MustActivateNextUpcomingActivity() {
		hostID, cmdUUID := aa.ActivateNextUpcomingActivityArgs()
		if err := s.providers.ActivateNextUpcomingActivity(ctx, hostID, cmdUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next upcoming activity")
		}
	}

	// Mark context as webhook processed
	ctx = context.WithValue(ctx, types.ActivityWebhookContextKey, true)

	return s.store.NewActivity(ctx, user, activity, detailsBytes, timestamp)
}

// detailsWithHostIDs returns the details to use in one per-fleet host
// activities webhook delivery: when the stored details carry no host
// identifier of their own (no host_id and no non-empty host_ids), it returns
// a copy with the delivery's host_ids added. On any failure it falls back to
// the stored details rather than blocking webhook delivery.
func (s *Service) detailsWithHostIDs(ctx context.Context, activityName string, detailsBytes []byte, hostIDs []uint) []byte {
	if len(hostIDs) == 0 {
		return detailsBytes
	}

	var details map[string]json.RawMessage
	if err := json.Unmarshal(detailsBytes, &details); err != nil {
		s.logger.ErrorContext(ctx, "unmarshal details to inject host_ids",
			slog.String("activity", activityName), slog.String("err", err.Error()))
		return detailsBytes
	}
	if _, ok := details["host_id"]; ok {
		return detailsBytes
	}
	// A stored host_ids list is kept only when it actually identifies hosts:
	// an empty or malformed one (e.g. a future type serializing an empty
	// HostIDList) is replaced with the delivery's list.
	if raw, ok := details["host_ids"]; ok {
		var stored []uint
		if err := json.Unmarshal(raw, &stored); err == nil && len(stored) > 0 {
			return detailsBytes
		}
	}
	if details == nil { // details were JSON null
		details = make(map[string]json.RawMessage, 1)
	}

	idsBytes, err := json.Marshal(hostIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "marshal host_ids",
			slog.String("activity", activityName), slog.String("err", err.Error()))
		return detailsBytes
	}
	details["host_ids"] = idsBytes
	withIDs, err := json.Marshal(details)
	if err != nil {
		s.logger.ErrorContext(ctx, "marshal details with host_ids",
			slog.String("activity", activityName), slog.String("err", err.Error()))
		return detailsBytes
	}
	return withIDs
}

// fireActivityWebhook sends the activity to the configured webhook URL asynchronously.
// It uses exponential backoff with a max elapsed time of 30 minutes for retries.
func (s *Service) fireActivityWebhook(
	ctx context.Context, user *api.User, activity api.ActivityDetails,
	detailsBytes []byte, timestamp time.Time, webhookURL string,
) {
	var userID *uint
	var userName *string
	var userEmail *string
	activityType := activity.ActivityName()

	if user != nil {
		// To support creating activities with users that were deleted. This can happen
		// for automatically installed software which uses the author of the upload as the author of
		// the installation.
		if user.ID != 0 && !user.Deleted {
			userID = &user.ID
		}
		userName = &user.Name
		userEmail = &user.Email
	} else if automatableActivity, ok := activity.(types.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
		automationAuthor := types.ActivityAutomationAuthor
		userName = &automationAuthor
	}

	// Capture the parent span for linking before launching the goroutine.
	parentSpanCtx := trace.SpanContextFromContext(ctx)

	go func() {
		// Create a root span for this async webhook delivery, linked back to the
		// originating request so traces can be correlated.
		var linkOpts []trace.SpanStartOption
		if parentSpanCtx.IsValid() {
			linkOpts = append(linkOpts, trace.WithLinks(trace.Link{SpanContext: parentSpanCtx}))
		}
		spanCtx, span := tracer.Start(
			context.Background(), "activity.webhook",
			append(linkOpts,
				trace.WithNewRoot(),
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(attribute.String("activity.type", activityType)),
			)...,
		)
		defer span.End()

		retryStrategy := backoff.NewExponentialBackOff()
		retryStrategy.MaxElapsedTime = 30 * time.Minute
		err := backoff.Retry(
			func() error {
				if err := platformhttp.PostJSONWithTimeout(
					spanCtx, webhookURL, &webhookPayload{
						Timestamp:     timestamp,
						ActorFullName: userName,
						ActorID:       userID,
						ActorEmail:    userEmail,
						Type:          activityType,
						Details:       (*json.RawMessage)(&detailsBytes),
					}, s.logger,
				); err != nil {
					var statusCoder kithttp.StatusCoder
					if errors.As(err, &statusCoder) && statusCoder.StatusCode() == http.StatusTooManyRequests {
						s.logger.DebugContext(spanCtx, "fire activity webhook", slog.String("err", err.Error()))
						return err
					}
					return backoff.Permanent(err)
				}
				return nil
			}, retryStrategy,
		)
		if err != nil {
			maskedErr := platformhttp.MaskURLError(err)
			span.RecordError(maskedErr)
			s.logger.ErrorContext(spanCtx,
				fmt.Sprintf("fire activity webhook to %s", platformhttp.MaskSecretURLParams(webhookURL)),
				slog.String("err", maskedErr.Error()),
			)
		}
	}()
}
