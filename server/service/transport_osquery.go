package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func decodeEnrollAgentRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req enrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return req, nil
}

func decodeSubmitLogsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var err error
	body := r.Body
	if r.Header.Get("content-encoding") == "gzip" {
		body, err = gzip.NewReader(body)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decoding gzip")
		}
		defer body.Close()
	}

	var req submitLogsRequest
	if err = json.NewDecoder(body).Decode(&req); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding JSON")
	}
	defer r.Body.Close()

	return req, nil
}
