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
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
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

func (r listActivitiesResponse) Error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
	_, err := newActivity(ctx, user, activity, svc.ds, svc.logger)

	return err
}

func (svc *Service) NewActivityWithHostLifecycleEvent(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, host *fleet.Host) (*fleet.HostLifecycleEvent, error) {
	return newActivityWithHostLifecycleEvent(ctx, user, activity, host, svc.ds, svc.logger)
}

func newActivityWithHostLifecycleEvent(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, host *fleet.Host, ds fleet.Datastore, logger kitlog.Logger) (*fleet.HostLifecycleEvent, error) {
	actID, err := newActivity(ctx, user, activity, ds, logger)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity")
	}

	// TODO: validate the activity is a host lifecycle event
	et := fleet.HostLifecycleEventType(activity.ActivityName())
	if !et.Valid() {
		return nil, ctxerr.Wrap(ctx, fmt.Errorf("invalid host lifecycle event type %q", et))
	}

	// TODO: populate the event
	hle := fleet.HostLifecycleEvent{
		ActivityID: &actID,
		HostSerial: host.HardwareSerial,
		HostUUID:   host.UUID,
		HostID:     host.ID,
		EventType:  et,
	}

	return ds.CreateHostLifecycleEvent(ctx, &hle)
}

var automationActivityAuthor = "Fleet"

func newActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, ds fleet.Datastore, logger kitlog.Logger) (uint, error) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get app config")
	}

	detailsBytes, err := json.Marshal(activity)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "marshaling activity details")
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
	Activities []*fleet.UpcomingActivity `json:"activities"`
	Count      uint                      `json:"count"`
	Err        error                     `json:"error,omitempty"`
}

func (r listHostUpcomingActivitiesResponse) Error() error { return r.Err }

func listHostUpcomingActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostUpcomingActivitiesRequest)
	acts, meta, err := svc.ListHostUpcomingActivities(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return listHostUpcomingActivitiesResponse{Err: err}, nil
	}

	return listHostUpcomingActivitiesResponse{Meta: meta, Activities: acts, Count: meta.TotalResults}, nil
}

// ListHostUpcomingActivities returns a slice of upcoming activities for the
// specified host.
func (svc *Service) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.UpcomingActivity, *fleet.PaginationMetadata, error) {
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

func listHostPastActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

////////////////////////////////////////////////////////////////////////////////
// Cancel host upcoming activity
////////////////////////////////////////////////////////////////////////////////

type cancelHostUpcomingActivityRequest struct {
	HostID     uint   `url:"id"`
	ActivityID string `url:"activity_id"`
}

type cancelHostUpcomingActivityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r cancelHostUpcomingActivityResponse) Error() error { return r.Err }
func (r cancelHostUpcomingActivityResponse) Status() int  { return http.StatusNoContent }

func cancelHostUpcomingActivityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*cancelHostUpcomingActivityRequest)
	err := svc.CancelHostUpcomingActivity(ctx, req.HostID, req.ActivityID)
	if err != nil {
		return cancelHostUpcomingActivityResponse{Err: err}, nil
	}
	return cancelHostUpcomingActivityResponse{}, nil
}

func (svc *Service) CancelHostUpcomingActivity(ctx context.Context, hostID uint, executionID string) error {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host")
	}
	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionCancelHostActivity); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	// prevent cancellation of lock/wipe that are already activated
	actMeta, err := svc.ds.GetHostUpcomingActivityMeta(ctx, hostID, executionID)
	if err != nil {
		return err
	}
	if actMeta.ActivatedAt != nil &&
		(actMeta.WellKnownAction == fleet.WellKnownActionLock || actMeta.WellKnownAction == fleet.WellKnownActionWipe) {
		return &fleet.BadRequestError{
			Message: "Couldn't cancel activity. Lock and wipe can't be canceled if they're about to run to prevent you from losing access to the host.",
		}
	}

	pastAct, err := svc.ds.CancelHostUpcomingActivity(ctx, hostID, executionID)
	if err != nil {
		return err
	}

	if pastAct != nil {
		if err := svc.NewActivity(ctx, vc.User, pastAct); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for cancelation")
		}
	}
	return nil
}
