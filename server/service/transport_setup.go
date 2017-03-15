package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeSetupRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
