package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeHostByIdentifierRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	identifier, err := nameFromRequest(r, "identifier")
	if err != nil {
		return nil, err
	}
	return hostByIdentifierRequest{Identifier: identifier}, nil
}

func decodeDeleteHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteHostRequest{ID: id}, nil
}

func decodeRefetchHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return refetchHostRequest{ID: id}, nil
}

func decodeAddHostsToTeamRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req addHostsToTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeAddHostsToTeamByFilterRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req addHostsToTeamByFilterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
