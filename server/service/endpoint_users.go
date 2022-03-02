package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

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
