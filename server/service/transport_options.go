package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
)

func decodeModifyOptionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req kolide.OptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
