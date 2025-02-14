package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/proxy"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Service struct {
	logger  kitlog.Logger
	authz   *authz.Authorizer
	ds      android.Datastore
	fleetDS fleet.Datastore
	proxy   *proxy.Proxy
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
	Url string `json:"android_enterprise_signup_url"`
	androidResponse
}

func androidEnterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	result, err := svc.EnterpriseSignup(ctx)
	if err != nil {
		return newErrResponse(err)
	}
	return androidEnterpriseSignupResponse{Url: result.Url}
}

func (svc *Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	appConfig, err := svc.fleetDS.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if appConfig.MDM.AndroidEnabledAndConfigured {
		return nil, fleet.NewInvalidArgumentError("android",
			"Android is already enabled and configured").WithStatus(http.StatusConflict)
	}

	id, err := svc.ds.CreateEnterprise(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	callbackURL := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/%d/connect", appConfig.ServerSettings.ServerURL, id)
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

func androidEnterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*androidEnterpriseSignupCallbackRequest)
	err := svc.EnterpriseSignupCallback(ctx, req.ID, req.EnterpriseToken)
	return androidResponse{Err: err}
}

func (svc *Service) EnterpriseSignupCallback(ctx context.Context, id uint, enterpriseToken string) error {
	// Skip authorization because the callback is called by Google.
	// TODO: Add some authorization here so random people can't bind random Android enterprises just for fun.
	svc.authz.SkipAuthorization(ctx)

	appConfig, err := svc.fleetDS.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	if appConfig.MDM.AndroidEnabledAndConfigured {
		return fleet.NewInvalidArgumentError("android",
			"Android is already enabled and configured").WithStatus(http.StatusConflict)
	}

	enterprise, err := svc.ds.GetEnterpriseByID(ctx, id)
	switch {
	case fleet.IsNotFound(err):
		return fleet.NewInvalidArgumentError("id",
			fmt.Sprintf("Enterprise with ID %d not found", id)).WithStatus(http.StatusNotFound)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	name, err := svc.proxy.EnterprisesCreate(
		[]string{"ENROLLMENT", "STATUS_REPORT", "COMMAND", "USAGE_LOGS"},
		enterpriseToken,
		enterprise.SignupName,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	enterpriseID := strings.TrimPrefix(name, "enterprises/")
	enterprise.EnterpriseID = enterpriseID
	err = svc.ds.UpdateEnterprise(ctx, enterprise)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	err = svc.ds.DeleteOtherEnterprises(ctx, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting temp enterprises")
	}

	err = svc.fleetDS.SetAndroidEnabledAndConfigured(ctx, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting android enabled and configured")
	}

	return nil
}

func androidDeleteEnterpriseEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	err := svc.DeleteEnterprise(ctx)
	return androidResponse{Err: err}
}

func (svc *Service) DeleteEnterprise(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionWrite); err != nil {
		return err
	}

	// Get enterprise
	enterprise, err := svc.ds.GetEnterprise(ctx)
	switch {
	case fleet.IsNotFound(err):
		// No enterprise to delete
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	default:
		err = svc.proxy.EnterpriseDelete(enterprise.EnterpriseID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting enterprise via Google API")
		}
	}

	err = svc.ds.DeleteEnterprises(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting enterprises")
	}

	err = svc.fleetDS.SetAndroidEnabledAndConfigured(ctx, false)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clearing android enabled and configured")
	}

	return nil
}

type androidEnrollmentTokenRequest struct {
	EnterpriseID uint `url:"id"`
}

type androidEnrollmentTokenResponse struct {
	*android.EnrollmentToken
	androidResponse
}

func androidEnrollmentTokenEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
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
