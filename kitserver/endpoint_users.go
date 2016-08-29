package kitserver

import (
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"

	"github.com/kolide/kolide-ose/kolide"
)

type createUserRequest struct {
	payload kolide.UserPayload
}

type createUserResponse struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	Enabled            bool   `json:"enabled"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
	Err                error  `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func makeCreateUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserRequest)
		user, err := svc.NewUser(ctx, req.payload)
		if err != nil {
			return createUserResponse{Err: err}, nil
		}
		return createUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Email:              user.Email,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
		}, nil
	}
}

type getUserRequest struct {
	ID uint `json:"id"`
}

type getUserResponse struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	Enabled            bool   `json:"enabled"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
	Err                error  `json:"error,omitempty"`
}

func (r getUserResponse) error() error { return r.Err }

func makeGetUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserRequest)

		// TODO call Service
		// user, err := svc.NewUser(user)

		var user *kolide.User
		var err error
		_ = req

		return getUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Email:              user.Email,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
			Err:                err,
		}, nil
	}
}

type modifyUserRequest struct {
	ID      uint
	payload kolide.UserPayload
}

type modifyUserResponse struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	Enabled            bool   `json:"enabled"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
	Err                error  `json:"error,omitempty"`
}

func (r modifyUserResponse) error() error { return r.Err }

func makeModifyUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyUserRequest)

		// TODO call Service with UserPayload

		var user *kolide.User
		var err error
		_ = req

		return modifyUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Email:              user.Email,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
			Err:                err,
		}, nil
	}
}
