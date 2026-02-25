package service

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
)

// DecodeRequest implement requestDecoder interface to take full control of decoding the request
type decodeCalendarWebhookRequest struct{}

func (decodeCalendarWebhookRequest) DecodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req fleet.CalendarWebhookRequest
	eventUUID, ok := mux.Vars(r)["event_uuid"]
	if !ok {
		return nil, endpointer.ErrBadRoute
	}
	unescaped, err := url.PathUnescape(eventUUID)
	if err != nil {
		return "", ctxerr.Wrap(r.Context(), err, "unescape value in path")
	}
	req.EventUUID = unescaped

	req.GoogleChannelID = r.Header.Get("X-Goog-Channel-Id")
	req.GoogleResourceState = r.Header.Get("X-Goog-Resource-State")

	return &req, nil
}

func calendarWebhookEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CalendarWebhookRequest)
	err := svc.CalendarWebhook(ctx, req.EventUUID, req.GoogleChannelID, req.GoogleResourceState)
	if err != nil {
		return fleet.CalendarWebhookResponse{Err: err}, err
	}

	resp := fleet.CalendarWebhookResponse{}
	return resp, nil
}

func (svc *Service) CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}
