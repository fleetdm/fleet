package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server"
	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get activities
////////////////////////////////////////////////////////////////////////////////

type listActivitiesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listActivitiesResponse struct {
	Meta       *fleet.PaginationMetadata `json:"meta"`
	Activities []*fleet.Activity         `json:"activities"`
	Err        error                     `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listActivitiesRequest)
	activities, metadata, err := svc.ListActivities(ctx, fleet.ListActivitiesOptions{
		ListOptions: req.ListOptions,
	})
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return listActivitiesResponse{Meta: metadata, Activities: activities}, nil
}

// ListActivities returns a slice of activities for the whole organization
func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}
	return svc.ds.ListActivities(ctx, opt)
}

type ActivityWebhookPayload struct {
	Timestamp     time.Time        `json:"timestamp"`
	ActorFullName *string          `json:"actor_full_name"`
	ActorID       *uint            `json:"actor_id"`
	ActorEmail    *string          `json:"actor_email"`
	Type          string           `json:"type"`
	Details       *json.RawMessage `json:"details"`
}

func (svc *Service) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	return newActivity(ctx, user, activity, svc.ds, svc.logger)
}

var automationActivityAuthor = "Fleet"

func newActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, ds fleet.Datastore, logger kitlog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
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
			if user.ID != 0 {
				userID = &user.ID
			}
			userName = &user.Name
			userEmail = &user.Email
		} else if automatableActivity, ok := activity.(fleet.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
			userName = &automationActivityAuthor
		}

		go func() {
			retryStrategy := backoff.NewExponentialBackOff()
			retryStrategy.MaxElapsedTime = 30 * time.Minute
			err := backoff.Retry(
				func() error {
					if err := server.PostJSONWithTimeout(
						context.Background(), webhookURL, &ActivityWebhookPayload{
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
							level.Debug(logger).Log("msg", "fire activity webhook", "err", err)
							return err
						}
						return backoff.Permanent(err)
					}
					return nil
				}, retryStrategy,
			)
			if err != nil {
				level.Error(logger).Log(
					"msg", fmt.Sprintf("fire activity webhook to %s", server.MaskSecretURLParams(webhookURL)), "err",
					server.MaskURLError(err).Error(),
				)
			}
		}()
	}
	// We update the context to indicate that we processed the webhook.
	ctx = context.WithValue(ctx, fleet.ActivityWebhookContextKey, true)
	return ds.NewActivity(ctx, user, activity, detailsBytes, timestamp)
}

////////////////////////////////////////////////////////////////////////////////
// List host upcoming activities
////////////////////////////////////////////////////////////////////////////////

type listHostUpcomingActivitiesRequest struct {
	HostID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listHostUpcomingActivitiesResponse struct {
	Meta       *fleet.PaginationMetadata `json:"meta"`
	Activities []*fleet.Activity         `json:"activities"`
	Count      uint                      `json:"count"`
	Err        error                     `json:"error,omitempty"`
}

func (r listHostUpcomingActivitiesResponse) error() error { return r.Err }

func listHostUpcomingActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostUpcomingActivitiesRequest)
	acts, meta, err := svc.ListHostUpcomingActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listHostUpcomingActivitiesResponse{Err: err}, nil
	}

	return listHostUpcomingActivitiesResponse{Meta: meta, Activities: acts, Count: meta.TotalResults}, nil
}

// ListHostUpcomingActivities returns a slice of upcoming activities for the
// specified host.
func (svc *Service) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, nil, err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host")
	}
	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported for upcoming activities
	opt.After = ""
	// custom ordering is not supported, always by upcoming queue order
	// (acual order is in the query, not set via ListOptions)
	opt.OrderKey = ""
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata
	opt.IncludeMetadata = true

	return svc.ds.ListHostUpcomingActivities(ctx, hostID, opt)
}

////////////////////////////////////////////////////////////////////////////////
// List host past activities
////////////////////////////////////////////////////////////////////////////////

type listHostPastActivitiesRequest struct {
	HostID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

func listHostPastActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostPastActivitiesRequest)
	acts, meta, err := svc.ListHostPastActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return &listActivitiesResponse{Meta: meta, Activities: acts}, nil
}

func (svc *Service) ListHostPastActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, nil, err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host")
	}
	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported for past activities
	opt.After = ""
	// custom ordering is not supported, always by date (newest first)
	opt.OrderKey = "created_at"
	opt.OrderDirection = fleet.OrderDescending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata
	opt.IncludeMetadata = true

	return svc.ds.ListHostPastActivities(ctx, hostID, opt)
}
