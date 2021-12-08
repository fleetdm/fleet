package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeModifyScheduledQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyScheduledQueryRequest

	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	req.ID = uint(id)
	return req, nil
}

func decodeDeleteScheduledQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteScheduledQueryRequest
	req.ID = uint(id)
	return req, nil
}
