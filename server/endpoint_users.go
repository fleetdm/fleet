package server

import (
	"net/http"

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
	ID                       uint   `json:"id"`
	Username                 string `json:"username"`
	Email                    string `json:"email"`
	Name                     string `json:"name"`
	Admin                    bool   `json:"admin"`
	Enabled                  bool   `json:"enabled"`
	Position                 string `json:"position,omitempty"`
	AdminForcedPasswordReset bool   `json:"force_password_reset"`
	Err                      error  `json:"error,omitempty"`
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
			ID:                       user.ID,
			Username:                 user.Username,
			Email:                    user.Email,
			Admin:                    user.Admin,
			Enabled:                  user.Enabled,
			Position:                 user.Position,
			AdminForcedPasswordReset: user.AdminForcedPasswordReset,
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
	ID                       uint   `json:"id"`
	Username                 string `json:"username"`
	Email                    string `json:"email"`
	Name                     string `json:"name"`
	Admin                    bool   `json:"admin"`
	Enabled                  bool   `json:"enabled"`
	Position                 string `json:"position,omitempty"`
	AdminForcedPasswordReset bool   `json:"force_password_reset"`
	Err                      error  `json:"error,omitempty"`
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
			ID:                       user.ID,
			Username:                 user.Username,
			Email:                    user.Email,
			Admin:                    user.Admin,
			Enabled:                  user.Enabled,
			Position:                 user.Position,
			AdminForcedPasswordReset: user.AdminForcedPasswordReset,
		}, nil
	}
}

func makeGetSessionUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := svc.AuthenticatedUser(ctx)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
		return getUserResponse{
			ID:                       user.ID,
			Username:                 user.Username,
			Email:                    user.Email,
			Admin:                    user.Admin,
			Enabled:                  user.Enabled,
			Position:                 user.Position,
			AdminForcedPasswordReset: user.AdminForcedPasswordReset,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Users
////////////////////////////////////////////////////////////////////////////////

type listUsersResponse struct {
	Users []getUserResponse `json:"users"`
	Err   error             `json:"error,omitempty"`
}

func (r listUsersResponse) error() error { return r.Err }

func makeListUsersEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		users, err := svc.Users(ctx)
		if err != nil {
			return listUsersResponse{Err: err}, nil
		}

		var resp listUsersResponse
		for _, user := range users {
			resp.Users = append(resp.Users, getUserResponse{
				ID:                       user.ID,
				Username:                 user.Username,
				Email:                    user.Email,
				Admin:                    user.Admin,
				Enabled:                  user.Enabled,
				Position:                 user.Position,
				AdminForcedPasswordReset: user.AdminForcedPasswordReset,
			})
		}
		return resp, nil
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
	ID                       uint   `json:"id"`
	Username                 string `json:"username"`
	Email                    string `json:"email"`
	Name                     string `json:"name"`
	Admin                    bool   `json:"admin"`
	Enabled                  bool   `json:"enabled"`
	Position                 string `json:"position,omitempty"`
	AdminForcedPasswordReset bool   `json:"force_password_reset"`
	Err                      error  `json:"error,omitempty"`
}

func (r modifyUserResponse) error() error { return r.Err }

func makeModifyUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyUserRequest)
		user, err := svc.ModifyUser(ctx, req.ID, req.payload)
		if err != nil {
			return modifyUserResponse{Err: err}, nil
		}

		return modifyUserResponse{
			ID:                       user.ID,
			Username:                 user.Username,
			Email:                    user.Email,
			Admin:                    user.Admin,
			Enabled:                  user.Enabled,
			Position:                 user.Position,
			AdminForcedPasswordReset: user.AdminForcedPasswordReset,
			Err: err,
		}, nil
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
		err := svc.RequestPasswordReset(ctx, req.Email)
		if err != nil {
			return forgotPasswordResponse{Err: err}, nil
		}
		return forgotPasswordResponse{}, nil
	}
}
