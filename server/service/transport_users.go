package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func decodeCreateUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodePerformRequiredPasswordResetRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req performRequiredPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding JSON")
	}
	return req, nil
}

func decodeForgotPasswordRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeResetPasswordRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
