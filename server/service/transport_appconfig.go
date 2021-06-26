package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func decodeModifyAppConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var payload fleet.AppConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return appConfigRequest{Payload: payload}, nil
}

func decodeApplyEnrollSecretSpecRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyEnrollSecretSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}
