package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Create User With Invite
////////////////////////////////////////////////////////////////////////////////

type createUserRequest struct {
	payload kolide.UserPayload
}

type createUserResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func makeCreateUserWithInviteEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserRequest)
		user, err := svc.CreateUserWithInvite(ctx, req.payload)
		if err != nil {
			return createUserResponse{Err: err}, nil
		}
		return createUserResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Create User
////////////////////////////////////////////////////////////////////////////////

func makeCreateUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserRequest)
		user, err := svc.CreateUser(ctx, req.payload)
		if err != nil {
			return createUserResponse{Err: err}, nil
		}
		return createUserResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get User
////////////////////////////////////////////////////////////////////////////////

type getUserRequest struct {
	ID uint `json:"id"`
}

type getUserResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r getUserResponse) error() error { return r.Err }

func makeGetUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserRequest)
		user, err := svc.User(ctx, req.ID)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
		return getUserResponse{User: user}, nil
	}
}

type adminUserRequest struct {
	ID    uint `json:"id"`
	Admin bool `json:"admin"`
}

type adminUserResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r adminUserResponse) error() error { return r.Err }

func makeAdminUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(adminUserRequest)
		user, err := svc.ChangeUserAdmin(ctx, req.ID, req.Admin)
		if err != nil {
			return adminUserResponse{Err: err}, nil
		}
		return adminUserResponse{User: user}, nil
	}
}

type enableUserRequest struct {
	ID      uint `json:"id"`
	Enabled bool `json:"enabled"`
}

type enableUserResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r enableUserResponse) error() error { return r.Err }

func makeEnableUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(enableUserRequest)
		user, err := svc.ChangeUserEnabled(ctx, req.ID, req.Enabled)
		if err != nil {
			return enableUserResponse{Err: err}, nil
		}
		return enableUserResponse{User: user}, nil
	}
}

func makeGetSessionUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := svc.AuthenticatedUser(ctx)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
		return getUserResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Users
////////////////////////////////////////////////////////////////////////////////

type listUsersRequest struct {
	ListOptions kolide.ListOptions
}

type listUsersResponse struct {
	Users []kolide.User `json:"users"`
	Err   error         `json:"error,omitempty"`
}

func (r listUsersResponse) error() error { return r.Err }

func makeListUsersEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUsersRequest)
		users, err := svc.ListUsers(ctx, req.ListOptions)
		if err != nil {
			return listUsersResponse{Err: err}, nil
		}

		resp := listUsersResponse{Users: []kolide.User{}}
		for _, user := range users {
			resp.Users = append(resp.Users, *user)
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Change Password
////////////////////////////////////////////////////////////////////////////////

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type changePasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r changePasswordResponse) error() error { return r.Err }

func makeChangePasswordEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changePasswordRequest)
		err := svc.ChangePassword(ctx, req.OldPassword, req.NewPassword)
		return changePasswordResponse{Err: err}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Reset Password
////////////////////////////////////////////////////////////////////////////////

type resetPasswordRequest struct {
	PasswordResetToken string `json:"password_reset_token"`
	NewPassword        string `json:"new_password"`
}

type resetPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resetPasswordResponse) error() error { return r.Err }

func makeResetPasswordEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetPasswordRequest)
		err := svc.ResetPassword(ctx, req.PasswordResetToken, req.NewPassword)
		return resetPasswordResponse{Err: err}, nil
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
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r modifyUserResponse) error() error { return r.Err }

func makeModifyUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyUserRequest)
		user, err := svc.ModifyUser(ctx, req.ID, req.payload)
		if err != nil {
			return modifyUserResponse{Err: err}, nil
		}

		return modifyUserResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Perform Required Password Reset
////////////////////////////////////////////////////////////////////////////////

type performRequiredPasswordResetRequest struct {
	Password string `json:"new_password"`
	ID       uint   `json:"id"`
}

type performRequiredPasswordResetResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r performRequiredPasswordResetResponse) error() error { return r.Err }

func makePerformRequiredPasswordResetEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(performRequiredPasswordResetRequest)
		user, err := svc.PerformRequiredPasswordReset(ctx, req.Password)
		if err != nil {
			return performRequiredPasswordResetResponse{Err: err}, nil
		}
		return performRequiredPasswordResetResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Require Password Reset
////////////////////////////////////////////////////////////////////////////////

type requirePasswordResetRequest struct {
	Require bool `json:"require"`
	ID      uint `json:"id"`
}

type requirePasswordResetResponse struct {
	User *kolide.User `json:"user,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r requirePasswordResetResponse) error() error { return r.Err }

func makeRequirePasswordResetEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(requirePasswordResetRequest)
		user, err := svc.RequirePasswordReset(ctx, req.ID, req.Require)
		if err != nil {
			return requirePasswordResetResponse{Err: err}, nil
		}
		return requirePasswordResetResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Forgot Password
////////////////////////////////////////////////////////////////////////////////

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type forgotPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r forgotPasswordResponse) error() error { return r.Err }
func (r forgotPasswordResponse) status() int  { return http.StatusAccepted }

func makeForgotPasswordEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(forgotPasswordRequest)
		// Any error returned by the service should not be returned to the
		// client to prevent information disclosure (it will be logged in the
		// server logs).
		_ = svc.RequestPasswordReset(ctx, req.Email)
		return forgotPasswordResponse{}, nil
	}
}
