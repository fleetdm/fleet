package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func decodeDownloadCarveRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	name, err := nameFromRequest(r, "name")
	if err != nil {
		return nil, err
	}
	return downloadCarveRequest{Name: name}, nil
}

func encodeDownloadCarveResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	resp, ok := response.(downloadCarveResponse)
	if !ok {
		encodeError(ctx, fmt.Errorf("expected carve response"), w)
		return nil
	}

	_, err := io.Copy(w, resp.Reader)
	return err
}
