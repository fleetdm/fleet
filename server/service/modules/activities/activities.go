package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type activityModule struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

type ActivityModule interface {
	NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error
}

func NewActivityModule(ds fleet.Datastore, logger kitlog.Logger) ActivityModule {
	return &activityModule{
		ds:     ds,
		logger: logger,
	}
}

var automationActivityAuthor = "Fleet"

func (a *activityModule) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	appConfig, err := a.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}

	detailsBytes, err := json.Marshal(activity)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling activity details")
	}
	timestamp := time.Now()

	if appConfig.WebhookSettings.ActivitiesWebhook.Enable {
		webhookURL := appConfig.WebhookSettings.ActivitiesWebhook.DestinationURL
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
		} else if automatableActivity, ok := activity.(fleet.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
			userName = &automationActivityAuthor
		}

		// TODO: webhook module? probably webhook job too tbh since this isn't very resilient
		go func() {
			retryStrategy := backoff.NewExponentialBackOff()
			retryStrategy.MaxElapsedTime = 30 * time.Minute
			err := backoff.Retry(
				func() error {
					if err := server.PostJSONWithTimeout(
						context.Background(), webhookURL, &fleet.ActivityWebhookPayload{
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
							level.Debug(a.logger).Log("msg", "fire activity webhook", "err", err)
							return err
						}
						return backoff.Permanent(err)
					}
					return nil
				}, retryStrategy,
			)
			if err != nil {
				level.Error(a.logger).Log(
					"msg", fmt.Sprintf("fire activity webhook to %s", server.MaskSecretURLParams(webhookURL)), "err",
					server.MaskURLError(err).Error(),
				)
			}
		}()
	}
	// We update the context to indicate that we processed the webhook.
	ctx = context.WithValue(ctx, fleet.ActivityWebhookContextKey, true)
	return a.ds.NewActivity(ctx, user, activity, detailsBytes, timestamp)
}
