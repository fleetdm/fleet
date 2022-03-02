package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

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
