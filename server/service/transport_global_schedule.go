package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeGetGlobalScheduleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opts, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	var req getGlobalScheduleRequest
	req.ListOptions = opts
	return req, nil
}

func decodeModifyGlobalScheduleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyGlobalScheduleRequest

	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	req.ID = id
	return req, nil
}

func decodeDeleteGlobalScheduleRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteGlobalScheduleRequest
	req.ID = id
	return req, nil
}

func decodeGlobalScheduleQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req globalScheduleQueryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}
