package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-kit/log/level"
	"google.golang.org/api/pubsub/v1"
)

type androidPubSubPushRequest struct {
}

// Compile-time check.
var _ endpoint_utils.RequestDecoder = androidPubSubPushRequest{}

func (androidPubSubPushRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {

	// TODO: Do full authentication
	if token, ok := r.URL.Query()["token"]; !ok || len(token) != 1 || token[0] == "" {
		return nil, fleet.NewAuthFailedError("missing token")
	}
	defer r.Body.Close()

	// TODO: Use our own struct for the message
	var message pubsub.PubsubMessage
	err := json.UnmarshalRead(r.Body, message)
	if err != nil {
		return nil, fleet.NewInvalidArgumentError("json", "invalid JSON")
	}

	return &message, nil
}

func androidPubSubPushEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*pubsub.PubsubMessage)
	err := svc.ProcessPubSubPush(ctx, req)
	return androidResponse{Err: err}
}

func (svc *Service) ProcessPubSubPush(ctx context.Context, message *pubsub.PubsubMessage) error {
	svc.authz.SkipAuthorization(ctx)
	level.Info(svc.logger).Log("msg", "Received PubSub push", "message", message)
	return nil
}
