package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

func decodeCarveBeginRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req carveBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(err, "decoding JSON")
	}

	return req, nil
}

func decodeCarveBlockRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req carveBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(err, "decoding JSON")
	}

	return req, nil
}

func decodeListCarvesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listCarvesRequest{ListOptions: opt}, nil
}
