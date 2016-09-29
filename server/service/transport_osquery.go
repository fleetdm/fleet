package service

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

func decodeEnrollAgentRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req enrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeGetClientConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req getClientConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeGetDistributedQueriesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req getDistributedQueriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeSubmitDistributedQueryResultsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req submitDistributedQueryResultsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeSubmitLogsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req submitLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}
