package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/android/interfaces"
	"github.com/fleetdm/fleet/v4/server/android/service/proxy"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet/common"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Service struct {
	logger  kitlog.Logger
	authz   *authz.Authorizer
	ds      android.Datastore
	fleetDS interfaces.FleetDatastore
	proxy   *proxy.Proxy
}

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	ds android.Datastore,
	fleetDS interfaces.FleetDatastore,
) (android.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	prx := proxy.NewProxy(ctx, logger)

	return &Service{
		logger:  logger,
		authz:   authorizer,
		ds:      ds,
		fleetDS: fleetDS,
		proxy:   prx,
	}, nil
}

type androidResponse struct {
	Err error `json:"error,omitempty"`
}

func (r androidResponse) Error() error { return r.Err }

func newErrResponse(err error) androidResponse {
	return androidResponse{Err: err}
}

type androidEnterpriseSignupResponse struct {
	*android.SignupDetails
	androidResponse
}

func androidEnterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc android.Service) errorer {
	result, err := svc.EnterpriseSignup(ctx)
	if err != nil {
		return newErrResponse(err)
	}
	return androidEnterpriseSignupResponse{SignupDetails: result}
}

func (svc *Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, common.ActionWrite); err != nil {
		return nil, err
	}

	appConfig, err := svc.fleetDS.CommonAppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	id, err := svc.ds.CreateEnterprise(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	callbackURL := fmt.Sprintf("%svc/api/v1/fleet/android_enterprise/%d/connect", appConfig.ServerURL(), id)
	signupDetails, err := svc.proxy.SignupURLsCreate(callbackURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating signup url")
	}

	err = svc.ds.UpdateEnterprise(ctx, &android.Enterprise{
		ID:         id,
		SignupName: signupDetails.Name,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	return signupDetails, nil
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

func (svc *Service) EnterpriseSignupCallback(ctx context.Context, id uint, enterpriseToken string) error {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, common.ActionWrite); err != nil {
		return err
	}

	// TODO: remove me
	level.Warn(svc.logger).Log("msg", "EnterpriseSignupCallback called", "id", id, "enterpriseToken", enterpriseToken)
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

func (svc *Service) CreateEnrollmentToken(ctx context.Context) (*android.EnrollmentToken, error) {
	svc.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(svc.logger).Log("msg", "CreateEnrollmentToken called")
	return nil, errors.New("not implemented")
}
