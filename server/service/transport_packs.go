package service

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

func decodeCreatePackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeModifyPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyPackRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}

func decodeDeletePackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deletePackRequest
	req.ID = id
	return req, nil
}

func decodeGetPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getPackRequest
	req.ID = id
	return req, nil
}

func decodeAddQueryToPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	qid, err := idFromRequest(r, "qid")
	if err != nil {
		return nil, err
	}
	pid, err := idFromRequest(r, "pid")
	if err != nil {
		return nil, err
	}
	var req addQueryToPackRequest
	req.PackID = pid
	req.QueryID = qid
	return req, nil
}

func decodeGetQueriesInPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getQueriesInPackRequest
	req.ID = id
	return req, nil
}

func decodeDeleteQueryFromPackRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	qid, err := idFromRequest(r, "qid")
	if err != nil {
		return nil, err
	}
	pid, err := idFromRequest(r, "pid")
	if err != nil {
		return nil, err
	}
	var req deleteQueryFromPackRequest
	req.PackID = pid
	req.QueryID = qid
	return req, nil
}
