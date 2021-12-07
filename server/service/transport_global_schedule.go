package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeModifyGlobalScheduleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyGlobalScheduleRequest

	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	req.ID = uint(id)
	return req, nil
}

func decodeDeleteGlobalScheduleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteGlobalScheduleRequest
	req.ID = uint(id)
	return req, nil
}
