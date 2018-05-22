package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeCreateDistributedQueryCampaignRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createDistributedQueryCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeCreateDistributedQueryCampaignByNamesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createDistributedQueryCampaignByNamesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
