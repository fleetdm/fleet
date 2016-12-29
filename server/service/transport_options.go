package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func decodeModifyOptionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req kolide.OptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
