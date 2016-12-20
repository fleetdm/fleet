package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/kolide-ose/server/kolide"

	"golang.org/x/net/context"
)

func decodeModifyAppConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req kolide.AppConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
