package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeDeletePackByIDRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deletePackByIDRequest
	req.ID = uint(id)
	return req, nil
}

func decodeGetPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getPackRequest
	req.ID = uint(id)
	return req, nil
}

func decodeApplyPackSpecsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyPackSpecsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}
