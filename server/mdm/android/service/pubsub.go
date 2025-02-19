package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-json-experiment/json"
	"github.com/go-kit/log/level"
)

type pubSubPushRequest struct {
	android.PubSubMessage `json:"message"`
	token                 string
}

// Compile-time check.
var _ endpoint_utils.RequestDecoder = pubSubPushRequest{}

func (pubSubPushRequest) DecodeRequest(_ context.Context, r *http.Request) (interface{}, error) {

	// TODO: Do full authentication
	var req pubSubPushRequest
	if tokens, ok := r.URL.Query()["token"]; !ok || len(tokens) != 1 || tokens[0] == "" {
		return nil, fleet.NewAuthFailedError("missing token")
	} else {
		req.token = tokens[0]
	}
	defer r.Body.Close()

	err := json.UnmarshalRead(r.Body, &req)
	if err != nil {
		return nil, fleet.NewInvalidArgumentError("json", fmt.Sprintf("invalid JSON: %s", err))
	}

	return &req, nil
}

func pubSubPushEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*pubSubPushRequest)
	err := svc.ProcessPubSubPush(ctx, &req.PubSubMessage)
	return defaultResponse{Err: err}
}

func (svc *Service) ProcessPubSubPush(ctx context.Context, message *android.PubSubMessage) error {
	svc.authz.SkipAuthorization(ctx)

	var rawData []byte
	if len(message.Data) > 0 {
		var err error
		rawData, err = base64.StdEncoding.DecodeString(message.Data)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "base64 decode message.data")
		}
	}

	switch message.Attributes["notificationType"] {
	case android.PubSubEnrollment:
		level.Warn(svc.logger).Log("msg", "Received PubSub enrollment", "message", fmt.Sprintf("%+v", rawData))
	}

	return nil
}
