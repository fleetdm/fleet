package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/kolide-ose/server/kolide"

	"golang.org/x/net/context"
)

func decodeModifyAppConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var payload kolide.AppConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return appConfigRequest{Payload: payload}, nil
}
