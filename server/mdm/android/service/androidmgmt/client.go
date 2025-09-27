package androidmgmt

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

// Client is used to interact with the Android Management API.
type Client interface {
	// SignupURLsCreate creates an enterprise signup URL.
	// See: https://developers.google.com/android/management/reference/rest/v1/signupUrls/create
	SignupURLsCreate(ctx context.Context, serverURL, callbackURL string) (*android.SignupDetails, error)

	// EnterprisesCreate creates an enterprise as well as the PubSub topic/subscription to receive notifications from Google.
	// This is the last step in the enterprise signup flow.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises/create
	// For PubSub integration, see: https://developers.google.com/android/management/notifications
	EnterprisesCreate(ctx context.Context, req EnterprisesCreateRequest) (EnterprisesCreateResponse, error)

	// EnterprisesPoliciesPatch updates or creates a policy.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/patch
	// On success it returns the applied policy, with its version number set.
	EnterprisesPoliciesPatch(ctx context.Context, policyName string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error)

	// EnterprisesDevicesPatch updates a device.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/patch
	// On success it returns the updated device with latest applied policy information.
	EnterprisesDevicesPatch(ctx context.Context, deviceName string, device *androidmanagement.Device) (*androidmanagement.Device, error)

	// EnterprisesDevicesDelete deletes an enrolled device (work profile) in the enterprise.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/delete
	EnterprisesDevicesDelete(ctx context.Context, deviceName string) error

	// EnterprisesEnrollmentTokensCreate creates an enrollment token for a given enterprise. It is used to enroll an Android device.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens/create
	EnterprisesEnrollmentTokensCreate(ctx context.Context, enterpriseName string,
		token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error)

	// EnterpriseDelete permanently deletes an enterprise and all accounts and data associated with it, including PubSub topic/subscription.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises/delete
	EnterpriseDelete(ctx context.Context, enterpriseName string) error

	// EnterprisesList lists all enterprises accessible to the calling user.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises/list
	EnterprisesList(ctx context.Context, serverURL string) ([]*androidmanagement.Enterprise, error)

	// SetAuthenticationSecret sets the secret used for authentication.
	SetAuthenticationSecret(secret string) error
}

type EnterprisesCreateRequest struct {
	// For Enterprise, EnterpriseToken, and SignupURLName details,
	// see: https://developers.google.com/android/management/reference/rest/v1/enterprises/create
	androidmanagement.Enterprise
	EnterpriseToken string
	SignupURLName   string

	// PubSubPushURL is the URL to push Android PubSub messages to.
	PubSubPushURL string
	// ServerURL is the Fleet server URL.
	ServerURL string
}

type EnterprisesCreateResponse struct {
	// EnterpriseName is the Google name of the Android Enterprise, like: enterprise/LC00r8aycu
	EnterpriseName string
	// FleetServerSecret is the secret used to authenticate with fleetdm.com. It is encrypted at rest.
	FleetServerSecret string
	// TopicName is the Google PubSub topic name, like: projects/project_id/topics/topic_id. It is only present Google API client is used
	// directly (no proxy). We save it for debugging purposes.
	TopicName string
}

// IsNotModifiedError reports whether the AMAPI error indicates that the
// resource has not been modified.
func IsNotModifiedError(err error) bool {
	return googleapi.IsNotModified(err)
}
