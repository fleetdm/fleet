package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func decodeGetCarveRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getCarveRequest{ID: int64(id)}, nil
}

func decodeListCarvesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	copt := fleet.CarveListOptions{ListOptions: opt}
	expired := r.URL.Query().Get("expired")
	switch expired {
	case "1", "true":
		copt.Expired = true
	case "0", "":
		copt.Expired = false
	default:
		return nil, errors.Errorf("invalid expired value %s", expired)
	}
	return listCarvesRequest{ListOptions: copt}, nil
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
