package service

import (
	"context"
	"encoding/json"
	"net/http"
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
	name, err := nameFromRequest(r, "name")
	if err != nil {
		return nil, err
	}
	var req deleteQueryRequest
	req.Name = name
	return req, nil
}

func decodeDeleteQueryByIDRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteQueryByIDRequest
	req.ID = id
	return req, nil
}

func decodeDeleteQueriesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req deleteQueriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
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

func decodeListQueriesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listQueriesRequest{ListOptions: opt}, nil
}

func decodeApplyQuerySpecsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyQuerySpecsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
