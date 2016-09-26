package service

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

func decodeCreateQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeModifyQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}

func decodeDeleteQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteQueryRequest
	req.ID = id
	return req, nil
}

func decodeGetQueryRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getQueryRequest
	req.ID = id
	return req, nil
}
