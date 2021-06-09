package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

func decodeCreateUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeGetUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getUserRequest{ID: id}, nil
}

func decodeListUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := userListOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listUsersRequest{ListOptions: opt}, nil
}

func decodeModifyUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req modifyUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}

func decodeDeleteUserRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteUserRequest{ID: id}, nil
}

func decodeChangePasswordRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeRequirePasswordResetRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, errors.Wrap(err, "getting ID from request")
	}

	var req requirePasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(err, "decoding JSON")
	}
	req.ID = id

	return req, nil
}

func decodePerformRequiredPasswordResetRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req performRequiredPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(err, "decoding JSON")
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
