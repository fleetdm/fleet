package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/android/interfaces"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type Service struct {
	logger  kitlog.Logger
	authz   *authz.Authorizer
	mgmt    *androidmanagement.Service
	ds      android.Datastore
	fleetDS interfaces.FleetDatastore
}

var (
	// Required env vars
	androidServiceCredentials = os.Getenv("FLEET_ANDROID_SERVICE_CREDENTIALS")
	androidProjectID          = os.Getenv("FLEET_ANDROID_PROJECT_ID")

	// Optional env vars
	androidPubSubTopic = os.Getenv("FLEET_ANDROID_PUBSUB_TOPIC")
)

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	ds android.Datastore,
	fleetDS fleet.Datastore,
) (android.Service, error) {
	// TODO: Android management service should only be created when needed.
	if androidServiceCredentials == "" || androidProjectID == "" {
		level.Error(logger).Log("msg",
			"FLEET_ANDROID_SERVICE_CREDENTIALS, FLEET_ANDROID_PROJECT_ID, and FLEET_ANDROID_PUBSUB_TOPIC environment variables must be set to use Android management")
		return nil, nil
	}
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating android management service")
	}
	return Service{
		logger:  logger,
		authz:   authorizer,
		mgmt:    mgmt,
		ds:      ds,
		fleetDS: fleetDS,
	}, nil
}

type androidResponse struct {
	Err error `json:"error,omitempty"`
}

func (r androidResponse) error() error { return r.Err }

type androidEnterpriseSignupResponse struct {
	*android.SignupDetails
	androidResponse
}

func androidEnterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc android.Service) (errorer, error) {
	result, err := svc.EnterpriseSignup(ctx)
	if err != nil {
		return androidResponse{Err: err}, nil
	}
	return androidEnterpriseSignupResponse{SignupDetails: result}, nil
}

func (s Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	s.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(s.logger).Log("msg", "EnterpriseSignup called")

	appConfig, err := s.fleetDS.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	id, err := s.ds.CreateEnterprise(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	callbackURL := fmt.Sprintf("%s/api/v1/fleet/android/enterprise/%d/callback", appConfig.ServerSettings.ServerURL, id)
	signupURL, err := s.mgmt.SignupUrls.Create().ProjectId(androidProjectID).CallbackUrl(callbackURL).Do()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating signup url")
	}

	err = s.ds.UpdateEnterprise(ctx, &android.Enterprise{
		ID:         id,
		SignupName: signupURL.Name,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	return &android.SignupDetails{
		Url:  signupURL.Url,
		Name: signupURL.Name,
	}, nil
}

type androidEnterpriseSignupCallbackRequest struct {
	ID              uint   `url:"id"`
	EnterpriseToken string `query:"enterpriseToken"`
}

func androidEnterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc android.Service) (errorer, error) {
	req := request.(*androidEnterpriseSignupCallbackRequest)
	err := svc.EnterpriseSignupCallback(ctx, req.ID, req.EnterpriseToken)
	return androidResponse{Err: err}, nil
}

func (s Service) EnterpriseSignupCallback(ctx context.Context, id uint, enterpriseToken string) error {
	s.authz.SkipAuthorization(ctx)

	// TODO: remove me
	level.Warn(s.logger).Log("msg", "EnterpriseSignupCallback called", "id", id, "enterpriseToken", enterpriseToken)

	enterprise, err := s.ds.GetEnterpriseByID(ctx, id)
	switch {
	case fleet.IsNotFound(err):
		return fleet.NewInvalidArgumentError("id",
			fmt.Sprintf("Enterprise with ID %d not found", id)).WithStatus(http.StatusNotFound)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	gEnterprise := &androidmanagement.Enterprise{
		EnabledNotificationTypes: []string{"ENROLLMENT", "STATUS_REPORT", "COMMAND", "USAGE_LOGS"},
		PubsubTopic:              androidPubSubTopic, // will be ignored if empty
	}
	gEnterprise, err = s.mgmt.Enterprises.Create(gEnterprise).ProjectId(androidProjectID).EnterpriseToken(enterpriseToken).SignupUrlName(enterprise.SignupName).Do()
	switch {
	case googleapi.IsNotModified(err):
		s.logger.Log("msg", "Android enterprise was already created", "enterprise_id", enterprise.EnterpriseID)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "creating enterprise via Google API")
	}

	enterpriseID := strings.TrimPrefix(gEnterprise.Name, "enterprises/")
	enterprise.EnterpriseID = enterpriseID
	err = s.ds.UpdateEnterprise(ctx, enterprise)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	level.Info(s.logger).Log("msg", "Enterprise created", "enterprise_id", enterpriseID)

	return nil
}

type androidPoliciesRequest struct {
	EnterpriseID uint `url:"id"`
}

func androidPoliciesEndpoint(ctx context.Context, request interface{}, svc android.Service) (errorer, error) {
	req := request.(*androidPoliciesRequest)
	err := svc.CreateOrUpdatePolicy(ctx, req.EnterpriseID)
	return androidResponse{Err: err}, nil
}

func (s Service) CreateOrUpdatePolicy(ctx context.Context, fleetEnterpriseID uint) error {
	s.authz.SkipAuthorization(ctx)

	enterprise, err := s.ds.GetEnterpriseByID(ctx, fleetEnterpriseID)
	switch {
	case fleet.IsNotFound(err):
		return fleet.NewInvalidArgumentError("id",
			fmt.Sprintf("Enterprise with ID %d not found", fleetEnterpriseID)).WithStatus(http.StatusNotFound)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	policyName := fmt.Sprintf("enterprises/%s/policies/default", enterprise.EnterpriseID)
	_, err = s.mgmt.Enterprises.Policies.Patch(policyName, &androidmanagement.Policy{
		CameraAccess: "CAMERA_ACCESS_DISABLED",
		StatusReportingSettings: &androidmanagement.StatusReportingSettings{
			ApplicationReportsEnabled:    true,
			DeviceSettingsEnabled:        true,
			SoftwareInfoEnabled:          true,
			MemoryInfoEnabled:            true,
			NetworkInfoEnabled:           true,
			DisplayInfoEnabled:           true,
			PowerManagementEventsEnabled: true,
			HardwareStatusEnabled:        true,
			SystemPropertiesEnabled:      true,
			ApplicationReportingSettings: &androidmanagement.ApplicationReportingSettings{
				IncludeRemovedApps: true,
			},
			CommonCriteriaModeEnabled: true,
		},
	}).Do()
	switch {
	case googleapi.IsNotModified(err):
		s.logger.Log("msg", "Android policy not modified", "enterprise_id", enterprise.EnterpriseID)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "creating or updating policy via Google API")
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

func androidEnrollmentTokenEndpoint(ctx context.Context, request interface{}, svc android.Service) (errorer, error) {
	req := request.(*androidEnrollmentTokenRequest)
	token, err := svc.CreateEnrollmentToken(ctx, req.EnterpriseID)
	if err != nil {
		return androidResponse{Err: err}, nil
	}
	return androidEnrollmentTokenResponse{EnrollmentToken: token}, nil
}

func (s Service) CreateEnrollmentToken(ctx context.Context, fleetEnterpriseID uint) (*android.EnrollmentToken, error) {
	s.authz.SkipAuthorization(ctx)
	enterprise, err := s.ds.GetEnterpriseByID(ctx, fleetEnterpriseID)
	switch {
	case fleet.IsNotFound(err):
		return nil, fleet.NewInvalidArgumentError("id",
			fmt.Sprintf("Enterprise with ID %d not found", fleetEnterpriseID)).WithStatus(http.StatusNotFound)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	token, err := s.mgmt.Enterprises.EnrollmentTokens.Create(enterprise.Name(), &androidmanagement.EnrollmentToken{
		AllowPersonalUsage: "PERSONAL_USAGE_ALLOWED",
		PolicyName:         enterprise.Name() + "/policies/default",
	}).Do()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enrollment token via Google API")
	}

	return &android.EnrollmentToken{
		Value: token.Value,
	}, nil
}
