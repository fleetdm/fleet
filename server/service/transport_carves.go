package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func decodeCarveBeginRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req carveBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding JSON")
	}

	return req, nil
}

func decodeCarveBlockRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	defer r.Body.Close()

	var req carveBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding JSON")
	}

	return req, nil
}

func decodeGetCarveBlockRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	blockId, err := idFromRequest(r, "block_id")
	if err != nil {
		return nil, err
	}
	return getCarveBlockRequest{ID: int64(id), BlockId: int64(blockId)}, nil
}
