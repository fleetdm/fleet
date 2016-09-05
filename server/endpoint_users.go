package server

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Create User
////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////
// Get User
////////////////////////////////////////////////////////////////////////////////

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
		user, err := svc.User(ctx, req.ID)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
		return getUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Email:              user.Email,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Change Password
////////////////////////////////////////////////////////////////////////////////

type changePasswordRequest struct {
	UserID             uint   `json:"user_id"`
	CurrentPassword    string `json:"current_password"`
	PasswordResetToken string `json:"password_reset_token"`
	NewPassword        string `json:"new_password"`
}

type changePasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r changePasswordResponse) error() error { return r.Err }

func makeChangePasswordEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changePasswordRequest)
		err := svc.ChangePassword(ctx, req.UserID, req.CurrentPassword, req.NewPassword)
		return changePasswordResponse{Err: err}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Update Admin Role
////////////////////////////////////////////////////////////////////////////////

type updateAdminRoleRequest struct {
	UserID uint `json:"user_id"`
	Admin  bool `json:"admin"`
}

type updateAdminRoleResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateAdminRoleResponse) error() error { return r.Err }

func makeUpdateAdminRoleEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateAdminRoleRequest)
		err := svc.UpdateAdminRole(ctx, req.UserID, req.Admin)
		return updateAdminRoleResponse{Err: err}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Update User Status
////////////////////////////////////////////////////////////////////////////////

type updateUserStatusRequest struct {
	UserID          uint   `json:"user_id"`
	Enabled         bool   `json:"enabled"`
	CurrentPassword string `json:"current_password"`
}

type updateUserStatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateUserStatusResponse) error() error { return r.Err }

func makeUpdateUserStatusEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserStatusRequest)
		err := svc.UpdateUserStatus(ctx, req.UserID, req.CurrentPassword, req.Enabled)
		return updateUserStatusResponse{Err: err}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify User
////////////////////////////////////////////////////////////////////////////////

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
