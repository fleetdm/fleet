package service

import (
	"context"
	"net/http"
)

func decodeGetScheduledQueriesInPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getScheduledQueriesInPackRequest
	req.ID = id
	return req, nil
}
