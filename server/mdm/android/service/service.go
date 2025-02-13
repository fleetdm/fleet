package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
)

type Service struct {
	logger  kitlog.Logger
	authz   *authz.Authorizer
	mgmt    *androidmanagement.Service
	ds      android.Datastore
	fleetDS fleet.Datastore
}

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	ds android.Datastore,
	fleetDS fleet.Datastore,
) (android.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	// mgmt, err := androidmanagement.NewService(ctx)
	// if err != nil {
	// 	return nil, ctxerr.Wrap(ctx, err, "creating android management service")
	// }
	return Service{
		logger:  logger,
		authz:   authorizer,
		mgmt:    nil,
		ds:      ds,
		fleetDS: fleetDS,
	}, nil
}

type androidResponse struct {
	Err error `json:"error,omitempty"`
}

func (r androidResponse) Error() error { return r.Err }

type androidEnterpriseSignupResponse struct {
	*android.SignupDetails
	androidResponse
}

func androidEnterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc android.Service) errorer {
	result, err := svc.EnterpriseSignup(ctx)
	if err != nil {
		return androidResponse{Err: err}
	}
	return androidEnterpriseSignupResponse{SignupDetails: result}
}

func (s Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	s.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(s.logger).Log("msg", "EnterpriseSignup called")
	return nil, errors.New("not implemented")

}

type androidEnterpriseSignupCallbackRequest struct {
	ID              uint   `url:"id"`
	EnterpriseToken string `query:"enterpriseToken"`
}

func androidEnterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc android.Service) errorer {
	req := request.(*androidEnterpriseSignupCallbackRequest)
	err := svc.EnterpriseSignupCallback(ctx, req.ID, req.EnterpriseToken)
	return androidResponse{Err: err}
}

func (s Service) EnterpriseSignupCallback(ctx context.Context, id uint, enterpriseToken string) error {
	s.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(s.logger).Log("msg", "EnterpriseSignupCallback called", "id", id, "enterpriseToken", enterpriseToken)
	return errors.New("not implemented")
}

type androidEnrollmentTokenRequest struct {
	EnterpriseID uint `url:"id"`
}

type androidEnrollmentTokenResponse struct {
	*android.EnrollmentToken
	androidResponse
}

func androidEnrollmentTokenEndpoint(ctx context.Context, request interface{}, svc android.Service) errorer {
	token, err := svc.CreateEnrollmentToken(ctx)
	if err != nil {
		return androidResponse{Err: err}
	}
	return androidEnrollmentTokenResponse{EnrollmentToken: token}
}

func (s Service) CreateEnrollmentToken(ctx context.Context) (*android.EnrollmentToken, error) {
	s.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(s.logger).Log("msg", "CreateEnrollmentToken called")
	return nil, errors.New("not implemented")
}
