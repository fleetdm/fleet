package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Create User With Invite
////////////////////////////////////////////////////////////////////////////////

type createUserRequest struct {
	payload fleet.UserPayload
}

type createUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func makeCreateUserFromInviteEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserRequest)
		user, err := svc.CreateUserFromInvite(ctx, req.payload)
		if err != nil {
			return createUserResponse{Err: err}, nil
		}
		return createUserResponse{User: user}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Create User
////////////////////////////////////////////////////////////////////////////////

func makeCreateUserEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r getUserResponse) error() error { return r.Err }

func makeGetUserEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserRequest)
		user, err := svc.User(ctx, req.ID)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
		return getUserResponse{User: user}, nil
	}
}

func makeGetSessionUserEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	ListOptions fleet.UserListOptions
}

type listUsersResponse struct {
	Users []fleet.User `json:"users"`
	Err   error        `json:"error,omitempty"`
}

func (r listUsersResponse) error() error { return r.Err }

func makeListUsersEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUsersRequest)
		users, err := svc.ListUsers(ctx, req.ListOptions)
		if err != nil {
			return listUsersResponse{Err: err}, nil
		}

		resp := listUsersResponse{Users: []fleet.User{}}
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

func makeChangePasswordEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeResetPasswordEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	payload fleet.UserPayload
}

type modifyUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r modifyUserResponse) error() error { return r.Err }

func makeModifyUserEndpoint(svc fleet.Service) endpoint.Endpoint {
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
// Delete User
////////////////////////////////////////////////////////////////////////////////

type deleteUserRequest struct {
	ID uint `json:"id"`
}

type deleteUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteUserResponse) error() error { return r.Err }

func makeDeleteUserEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteUserRequest)
		err := svc.DeleteUser(ctx, req.ID)
		if err != nil {
			return deleteUserResponse{Err: err}, nil
		}
		return deleteUserResponse{}, nil
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
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r performRequiredPasswordResetResponse) error() error { return r.Err }

func makePerformRequiredPasswordResetEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r requirePasswordResetResponse) error() error { return r.Err }

func makeRequirePasswordResetEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeForgotPasswordEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(forgotPasswordRequest)
		// Any error returned by the service should not be returned to the
		// client to prevent information disclosure (it will be logged in the
		// server logs).
		_ = svc.RequestPasswordReset(ctx, req.Email)
		return forgotPasswordResponse{}, nil
	}
}
