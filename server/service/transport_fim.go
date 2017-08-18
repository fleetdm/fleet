package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
)

func decodeModifyFIMRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var fimConfig kolide.FIMConfig
	if err := json.NewDecoder(r.Body).Decode(&fimConfig); err != nil {
		return nil, err
	}
	return fimConfig, nil
}
