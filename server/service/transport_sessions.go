package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/sso"
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
	req.Email = strings.ToLower(req.Email)
	return req, nil
}

func decodeInitiateSSORequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req initiateSSORequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func decodeCallbackSSORequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decode sso callback")
	}
	authResponse, err := sso.DecodeAuthResponse(r.FormValue("SAMLResponse"))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding sso callback")
	}
	return authResponse, nil
}
