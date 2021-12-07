package service

import (
	"context"
	"net/http"
)

func decodeGetPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getPackRequest
	req.ID = uint(id)
	return req, nil
}
