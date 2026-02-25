package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func decodeSetupRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req fleet.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
