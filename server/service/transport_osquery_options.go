package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeApplyOsqueryOptionsSpecRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyOsqueryOptionsSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}
