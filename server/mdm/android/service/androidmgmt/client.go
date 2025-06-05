package androidmgmt

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
)

// Client is used to interact with the Android Management API.
type Client interface {
	// SignupURLsCreate creates an enterprise signup URL.
	// See: https://developers.google.com/android/management/reference/rest/v1/signupUrls/create
	SignupURLsCreate(serverURL, callbackURL string) (*android.SignupDetails, error)

	// EnterprisesCreate creates an enterprise. This is the last step in the enterprise signup flow.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises/create
	EnterprisesCreate(ctx context.Context, req EnterprisesCreateRequest) (EnterprisesCreateResponse, error)

	// EnterprisesPoliciesPatch updates or creates a policy.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/patch
	EnterprisesPoliciesPatch(policyName string, policy *androidmanagement.Policy) error

	// EnterprisesEnrollmentTokensCreate creates an enrollment token for a given enterprise. It is used to enroll an Android device.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens/create
	EnterprisesEnrollmentTokensCreate(enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error)

	// EnterpriseDelete permanently deletes an enterprise and all accounts and data associated with it.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises/delete
	EnterpriseDelete(ctx context.Context, enterpriseID string) error

	// SetFleetServerSecret sets the secret used to authenticate with fleetdm.com.
	SetFleetServerSecret(secret string)
}

type EnterprisesCreateRequest struct {
	// For Enterprise, EnterpriseToken, and SignupUrlName details,
	// see: https://developers.google.com/android/management/reference/rest/v1/enterprises/create
	androidmanagement.Enterprise
	EnterpriseToken string
	SignupUrlName   string

	// PubSubPushURL is the URL to push Android PubSub messages to.
	PubSubPushURL string
	// ServerURL is the Fleet server URL.
	ServerURL string
}

type EnterprisesCreateResponse struct {
	// EnterpriseName is the Google name of the Android Enterprise, like: enterprise/LC00r8aycu
	EnterpriseName string
	// FleetServerSecret is the secret used to authenticate with fleetdm.com.
	FleetServerSecret string
	// TopicName is the Google PubSub topic name, like: projects/project_id/topics/topic_id. It is only present Google API is used
	// directly (no proxy). We save it for debugging purposes.
	TopicName string
}
