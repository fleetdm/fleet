package service

import (
	"context"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

type Service struct {
	logger kitlog.Logger
	authz  *authz.Authorizer
	mgmt   *androidmanagement.Service
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
		logger: logger,
		authz:  authz,
		mgmt:   mgmt,
	}, nil
}

// ID from DB
const callbackURL = "https://example.com/api/v1/fleet/android/enterprise/1/callback"

const androidSignupName = "signupUrls/XXX"

// From ENV vars
// const pubSubTopic = "projects/android-api-448119/topics/android"

func (s Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	s.authz.SkipAuthorization(ctx)
	level.Warn(s.logger).Log("msg", "EnterpriseSignup called")

	// TODO: We should cache it (use a single struct) so we don't have to create a new client every time -- singleton pattern?
	signupURL, err := s.mgmt.SignupUrls.Create().ProjectId(androidProjectID).CallbackUrl(callbackURL).Do()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating signup url")
	}

	// TODO: Save the name in the DB so we can reference it later.

	return &android.SignupDetails{
		Url:  signupURL.Url,
		Name: signupURL.Name,
	}, nil
}

func (s Service) EnterpriseSignupCallback(ctx context.Context, enterpriseID uint, enterpriseToken string) error {
	s.authz.SkipAuthorization(ctx)
	level.Warn(s.logger).Log("msg", "EnterpriseSignupCallback called", "enterpriseID", enterpriseID, "enterpriseToken", enterpriseToken)

	// TODO: Get the name from the DB so we can reference it here.

	enterprise := &androidmanagement.Enterprise{
		EnabledNotificationTypes: []string{"ENROLLMENT", "STATUS_REPORT", "COMMAND", "USAGE_LOGS"},
		PubsubTopic:              androidPubSubTopic,
	}
	enterprise, err := s.mgmt.Enterprises.Create(enterprise).ProjectId(androidProjectID).EnterpriseToken(enterpriseToken).SignupUrlName(androidSignupName).Do()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating enterprise")
	}
	level.Warn(s.logger).Log("msg", "Enterprise created", "enterprise", fmt.Sprintf("%+v", *enterprise))

	// Name is enterprises/LC01ojfdkn, where the last part is the enterprise ID.

	return nil
}
