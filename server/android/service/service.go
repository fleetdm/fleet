package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

type Service struct {
	logger  kitlog.Logger
	authz   *authz.Authorizer
	mgmt    *androidmanagement.Service
	ds      android.Datastore
	fleetDS fleet.Datastore
}

// Required env vars:
var (
	androidServiceCredentials = os.Getenv("FLEET_ANDROID_SERVICE_CREDENTIALS")
	androidProjectID          = os.Getenv("FLEET_ANDROID_PROJECT_ID")
	androidPubSubTopic        = os.Getenv("FLEET_ANDROID_PUBSUB_TOPIC")
)

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	authz *authz.Authorizer,
	ds android.Datastore,
	fleetDS fleet.Datastore,
) (android.Service, error) {
	// TODO: Android management service should only be created when needed.
	if androidServiceCredentials == "" || androidProjectID == "" || androidPubSubTopic == "" {
		level.Error(logger).Log("msg",
			"FLEET_ANDROID_SERVICE_CREDENTIALS, FLEET_ANDROID_PROJECT_ID, and FLEET_ANDROID_PUBSUB_TOPIC environment variables must be set to use Android management")
		return nil, nil
	}
	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating android management service")
	}
	return Service{
		logger:  logger,
		authz:   authz,
		mgmt:    mgmt,
		ds:      ds,
		fleetDS: fleetDS,
	}, nil
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
		PubsubTopic:              androidPubSubTopic,
	}
	gEnterprise, err = s.mgmt.Enterprises.Create(gEnterprise).ProjectId(androidProjectID).EnterpriseToken(enterpriseToken).SignupUrlName(enterprise.SignupName).Do()
	if err != nil {
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
