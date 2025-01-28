package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// //////////////////////////////////////////////////////////////////////////////
// Android management
// //////////////////////////////////////////////////////////////////////////////

type androidResponse struct {
	Err error `json:"error,omitempty"`
}

func (r androidResponse) error() error { return r.Err }

type androidEnterpriseSignupResponse struct {
	*android.SignupDetails
	androidResponse
}

func androidEnterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (errorer, error) {
	result, err := svc.Android().EnterpriseSignup(ctx)
	if err != nil {
		return androidResponse{Err: err}, nil
	}
	return androidEnterpriseSignupResponse{SignupDetails: result}, nil
}

type androidEnterpriseSignupCallbackRequest struct {
	ID              uint   `url:"id"`
	EnterpriseToken string `query:"enterpriseToken"`
}

func androidEnterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*androidEnterpriseSignupCallbackRequest)
	err := svc.Android().EnterpriseSignupCallback(ctx, req.ID, req.EnterpriseToken)
	return androidResponse{Err: err}, nil
}

type androidPoliciesRequest struct {
	EnterpriseID uint `url:"id"`
}

func androidPoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*androidPoliciesRequest)
	err := svc.Android().CreateOrUpdatePolicy(ctx, req.EnterpriseID)
	return androidResponse{Err: err}, nil
}

type androidEnrollmentTokenRequest struct {
	EnterpriseID uint `url:"id"`
}

type androidEnrollmentTokenResponse struct {
	*android.EnrollmentToken
	androidResponse
}

func androidEnrollmentTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*androidEnrollmentTokenRequest)
	token, err := svc.Android().CreateEnrollmentToken(ctx, req.EnterpriseID)
	if err != nil {
		return androidResponse{Err: err}, nil
	}
	return androidEnrollmentTokenResponse{EnrollmentToken: token}, nil
}
