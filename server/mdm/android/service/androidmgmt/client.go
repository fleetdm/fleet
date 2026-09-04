package androidmgmt

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	EnterprisesPoliciesPatch(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts PoliciesPatchOpts) (*androidmanagement.Policy, error)

	// EnterprisesDevicesPatch updates a device.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/patch
	// On success it returns the updated device with latest applied policy information.
	EnterprisesDevicesPatch(ctx context.Context, deviceName string, device *androidmanagement.Device) (*androidmanagement.Device, error)

	// EnterprisesDevicesGet retrieves a device by resource name.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/get
	EnterprisesDevicesGet(ctx context.Context, deviceName string) (*androidmanagement.Device, error)

	// EnterprisesDevicesDelete deletes an enrolled device (work profile) in the enterprise.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/delete
	EnterprisesDevicesDelete(ctx context.Context, deviceName string) error

	// EnterprisesDevicesIssueCommand issues a command (LOCK, RESET_PASSWORD, WIPE, ...) to a
	// device. Returns the Operation whose metadata wraps the Command; the device-side result
	// arrives later as a Pub/Sub COMMAND notification correlated by operation name. See:
	// https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand
	EnterprisesDevicesIssueCommand(ctx context.Context, deviceName string, command *androidmanagement.Command) (*androidmanagement.Operation, error)

	// EnterprisesDevicesOperationsGet fetches the current state of an Operation returned by
	// EnterprisesDevicesIssueCommand. It is the authoritative source for a command's outcome and lets
	// Fleet reconcile commands whose Pub/Sub COMMAND notification never arrived. operationName is the
	// full AMAPI resource name (enterprises/X/devices/Y/operations/Z). See:
	// https://developers.google.com/android/management/reference/rest/v1/enterprises.devices.operations/get
	EnterprisesDevicesOperationsGet(ctx context.Context, operationName string) (*androidmanagement.Operation, error)

	// EnterprisesDevicesListPartial lists devices for the given enterprise with partial fields.
	// Page size of 100 devices
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/list
	// Currently the devices has the following attributes:
	// Name
	EnterprisesDevicesListPartial(ctx context.Context, enterpriseName string, pageToken string) (*androidmanagement.ListDevicesResponse, error)

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

	EnterprisesApplications(ctx context.Context, enterpriseName, packageName string) (*androidmanagement.Application, error)

	// EnterprisesPoliciesModifyPolicyApplications adds or updates the given apps in the policy.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/modifyPolicyApplications
	EnterprisesPoliciesModifyPolicyApplications(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error)

	// EnterprisesPoliciesRemovePolicyApplications removes the given apps from the policy.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies/removePolicyApplications
	EnterprisesPoliciesRemovePolicyApplications(ctx context.Context, policyName string, packageNames []string) (*androidmanagement.Policy, error)

	// EnterprisesWebAppsCreate creates a web app in the enterprise.
	// See: https://developers.google.com/android/management/reference/rest/v1/enterprises.webApps/create
	EnterprisesWebAppsCreate(ctx context.Context, enterpriseName string, webApp *androidmanagement.WebApp) (*androidmanagement.WebApp, error)
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

// IsBadRequestError reports whether the AMAPI error indicates that the
// request was invalid due to a client error.
func IsBadRequestError(err error) bool {
	if ae, ok := errors.AsType[*googleapi.Error](err); ok {
		return ae.Code == http.StatusBadRequest
	}
	return false
}

// IsNotFoundError reports whether the AMAPI error indicates that the requested
// resource does not exist. AMAPI sometimes returns a 500 with "Requested entity
// was not found" instead of a proper 404.
func IsNotFoundError(err error) bool {
	if ae, ok := errors.AsType[*googleapi.Error](err); ok {
		return ae.Code == http.StatusNotFound ||
			(ae.Code == http.StatusInternalServerError && strings.Contains(ae.Error(), "Requested entity was not found"))
	}
	return false
}

// IsAuthenticationError reports whether the AMAPI error indicates that the
// request was rejected over credentials or access, rather than anything about
// the resource that was requested.
func IsAuthenticationError(err error) bool {
	if ae, ok := errors.AsType[*googleapi.Error](err); ok {
		return ae.Code == http.StatusUnauthorized || ae.Code == http.StatusForbidden
	}
	return false
}

// IsTooManyRequestsError reports whether the AMAPI error indicates that we
// exceeded the project's request quota.
func IsTooManyRequestsError(err error) bool {
	if ae, ok := errors.AsType[*googleapi.Error](err); ok {
		return ae.Code == http.StatusTooManyRequests
	}
	return false
}

// IsConflictError reports whether the AMAPI error indicates that the device
// state is incompatible with the requested operation.
func IsConflictError(err error) bool {
	if ae, ok := errors.AsType[*googleapi.Error](err); ok {
		return ae.Code == http.StatusConflict
	}
	return false
}

// FleetErrFromAMAPI maps a *googleapi.Error to the corresponding Fleet error
// type. Returns nil when err is nil or is not a *googleapi.Error,
// in which case the caller should fall through to its default error handling.
func FleetErrFromAMAPI(err error) error {
	ae, ok := errors.AsType[*googleapi.Error](err)
	if !ok {
		return nil
	}
	msg := ae.Message
	if msg == "" {
		msg = ae.Body
	}
	switch {
	case IsBadRequestError(err):
		return &fleet.BadRequestError{Message: msg, InternalErr: err}
	case IsNotFoundError(err):
		return &notFoundError{message: msg, internalErr: err}
	case IsConflictError(err):
		return &fleet.ConflictError{Message: msg}
	default:
		return nil
	}
}

// notFoundError implements the fleet IsNotFound interface so the HTTP layer
// returns 404.
type notFoundError struct {
	message     string
	internalErr error
}

func (e *notFoundError) Error() string {
	if e.message != "" {
		return e.message
	}
	return "not found"
}

func (e *notFoundError) IsNotFound() bool    { return true }
func (e *notFoundError) IsClientError() bool { return true }

// Internal implements the ErrWithInternal interface so the AMAPI error is logged
// without being exposed in the HTTP response.
func (e *notFoundError) Internal() string {
	if e.internalErr == nil {
		return ""
	}
	return e.internalErr.Error()
}

// Unwrap returns the error array form, which errors.Is/As still traverse but
// errors.Unwrap does not. The HTTP layer type-switches on ctxerr.Cause (which walks
// errors.Unwrap to the root), so a single-error Unwrap would hand it the *googleapi.Error
// instead of this type and the response would be a 500 rather than a 404. BadRequestError
// uses the same form for the same reason.
func (e *notFoundError) Unwrap() []error {
	if e.internalErr == nil {
		return nil
	}
	return []error{e.internalErr}
}
