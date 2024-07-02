package service

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/mux"
)

type calendarWebhookRequest struct {
	eventUUID           string
	googleChannelID     string
	googleResourceState string
}

// DecodeRequest implement requestDecoder interface to take full control of decoding the request
func (calendarWebhookRequest) DecodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req calendarWebhookRequest
	eventUUID, ok := mux.Vars(r)["event_uuid"]
	if !ok {
		return nil, errBadRoute
	}
	unescaped, err := url.PathUnescape(eventUUID)
	if err != nil {
		return "", ctxerr.Wrap(r.Context(), err, "unescape value in path")
	}
	req.eventUUID = unescaped

	req.googleChannelID = r.Header.Get("X-Goog-Channel-Id")
	req.googleResourceState = r.Header.Get("X-Goog-Resource-State")

	return &req, nil
}

type calendarWebhookResponse struct {
	Err error `json:"error,omitempty"`
}

func (r calendarWebhookResponse) error() error { return r.Err }

func calendarWebhookEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*calendarWebhookRequest)
	err := svc.CalendarWebhook(ctx, req.eventUUID, req.googleChannelID, req.googleResourceState)
	if err != nil {
		return calendarWebhookResponse{Err: err}, err
	}

	resp := calendarWebhookResponse{}
	return resp, nil
}

func (svc *Service) CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}
