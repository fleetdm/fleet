package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func decodeGetInfoAboutSessionRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getInfoAboutSessionRequest{ID: id}, nil
}

func decodeGetInfoAboutSessionsForUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getInfoAboutSessionsForUserRequest{ID: id}, nil
}

func decodeDeleteSessionRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteSessionRequest{ID: id}, nil
}

func decodeDeleteSessionsForUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteSessionsForUserRequest{ID: id}, nil
}

func decodeLoginRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	req.Username = strings.ToLower(req.Username)
	return req, nil
}
