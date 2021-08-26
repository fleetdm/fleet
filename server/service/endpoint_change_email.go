package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type changeEmailRequest struct {
	Token string
}

type changeEmailResponse struct {
	NewEmail string `json:"new_email"`
	Err      error  `json:"error,omitempty"`
}

func (r changeEmailResponse) error() error { return r.Err }

func makeChangeEmailEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeEmailRequest)
		newEmailAddress, err := svc.ChangeUserEmail(ctx, req.Token)
		if err != nil {
			return changeEmailResponse{Err: err}, nil
		}
		return changeEmailResponse{NewEmail: newEmailAddress}, nil
	}
}
