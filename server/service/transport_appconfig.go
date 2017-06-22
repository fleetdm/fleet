package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
)

func decodeModifyAppConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var payload kolide.AppConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return appConfigRequest{Payload: payload}, nil
}
