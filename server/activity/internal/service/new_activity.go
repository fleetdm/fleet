package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log/level"
)

// NewActivity creates a new activity record and fires the webhook if configured.
func (s *Service) NewActivity(ctx context.Context, user *api.User, activity api.ActivityDetails) error {
	detailsBytes, err := json.Marshal(activity)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling activity details")
	}
	timestamp := time.Now()

	// Fire webhook if enabled
	webhookConfig, err := s.providers.GetActivitiesWebhookConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get activities webhook config")
	}

	if webhookConfig != nil && webhookConfig.Enable {
		s.fireActivityWebhook(user, activity, detailsBytes, timestamp, webhookConfig.DestinationURL)
	}

	// Activate the next upcoming activity if requested by the activity type.
	// This is done before storing to avoid holding a DB transaction open during
	// potentially slow operations.
	if aa, ok := activity.(api.ActivityActivator); ok && aa.MustActivateNextUpcomingActivity() {
		hostID, cmdUUID := aa.ActivateNextUpcomingActivityArgs()
		if err := s.providers.ActivateNextUpcomingActivity(ctx, hostID, cmdUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next upcoming activity")
		}
	}

	// Mark context as webhook processed
	ctx = context.WithValue(ctx, api.ActivityWebhookContextKey, true)

	return s.store.NewActivity(ctx, user, activity, detailsBytes, timestamp)
}

// fireActivityWebhook sends the activity to the configured webhook URL asynchronously.
// It uses exponential backoff with a max elapsed time of 30 minutes for retries.
func (s *Service) fireActivityWebhook(user *api.User, activity api.ActivityDetails, detailsBytes []byte, timestamp time.Time, webhookURL string) {
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
	} else if automatableActivity, ok := activity.(api.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
		automationAuthor := api.ActivityAutomationAuthor
		userName = &automationAuthor
	}

	go func() {
		retryStrategy := backoff.NewExponentialBackOff()
		retryStrategy.MaxElapsedTime = 30 * time.Minute
		err := backoff.Retry(
			func() error {
				if err := s.providers.SendWebhookPayload(
					context.Background(), webhookURL, &api.WebhookPayload{
						Timestamp:     timestamp,
						ActorFullName: userName,
						ActorID:       userID,
						ActorEmail:    userEmail,
						Type:          activityType,
						Details:       (*json.RawMessage)(&detailsBytes),
					},
				); err != nil {
					var statusCoder kithttp.StatusCoder
					if errors.As(err, &statusCoder) && statusCoder.StatusCode() == http.StatusTooManyRequests {
						level.Debug(s.logger).Log("msg", "fire activity webhook", "err", err)
						return err
					}
					return backoff.Permanent(err)
				}
				return nil
			}, retryStrategy,
		)
		if err != nil {
			level.Error(s.logger).Log(
				"msg", fmt.Sprintf("fire activity webhook to %s", s.providers.MaskSecretURLParams(webhookURL)),
				"err", s.providers.MaskURLError(err).Error(),
			)
		}
	}()
}
