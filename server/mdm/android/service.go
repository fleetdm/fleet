package android

import (
	"context"
	"net/http"

	"google.golang.org/api/androidmanagement/v1"
)

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, signupToken string, enterpriseToken string) error
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	DeleteEnterprise(ctx context.Context) error
	EnterpriseSignupSSE(ctx context.Context) (chan string, error)

	// CreateEnrollmentToken creates an enrollment token for a new Android device.
	CreateEnrollmentToken(ctx context.Context, enrollSecret, idpUUID string) (*EnrollmentToken, error)
	ProcessPubSubPush(ctx context.Context, token string, message *PubSubMessage) error

	// UnenrollAndroidHost triggers unenrollment (work profile removal) for the given Android host ID.
	UnenrollAndroidHost(ctx context.Context, hostID uint) error

	EnterprisesApplications(ctx context.Context, enterpriseName, applicationID string) (*androidmanagement.Application, error)
	AddAppsToAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*MDMAndroidPolicyRequest, error)
	// SetAppsForAndroidPolicy sets the available apps for the given hosts' Android MDM policy to the given list of apps.
	// Note that unlike AddAppsToAndroidPolicy, this method replaces the existing app list with the given one, it is
	// not additive/PATCH semantics.
	SetAppsForAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error
	AddFleetAgentToAndroidPolicy(ctx context.Context, enterpriseName string, hostConfigs map[string]AgentManagedConfiguration) error
	BuildFleetAgentApplicationPolicy(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error)
	// BuildAndSendFleetAgentConfig builds the complete AgentManagedConfiguration for the given hosts
	// (including certificate templates) and sends it to the Android Management API.
	// This is the centralized function that should be used by all callers to avoid race conditions.
	// If skipHostsWithoutNewCerts is true, hosts that don't have new certificate templates to deliver
	// will be skipped.
	BuildAndSendFleetAgentConfig(ctx context.Context, enterpriseName string, hostUUIDs []string, skipHostsWithoutNewCerts bool) error
	EnableAppReportsOnDefaultPolicy(ctx context.Context) error
	MigrateToPerDevicePolicy(ctx context.Context) error
	PatchDevice(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (skip bool, apiErr error)
	PatchPolicy(ctx context.Context, policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (skip bool, err error)

	// verifyExistingEnterpriseIfAny checks if there's an existing enterprise in the database
	// and if so, verifies it still exists in Google API. If it doesn't exist, performs cleanup.
	// Returns fleet.IsNotFound error if enterprise was deleted, nil if no enterprise exists or verification passed.
	VerifyExistingEnterpriseIfAny(ctx context.Context) error
}

// /////////////////////////////////////////////
// Android API request and response structs

type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DefaultResponse) Error() error { return r.Err }

// StatusCode implements the go-kit http StatusCoder interface to preserve HTTP status codes from errors
func (r DefaultResponse) StatusCode() int {
	if r.Err != nil {
		// Check if the error has a custom status code (like errors created with .WithStatus())
		if sc, ok := r.Err.(interface{ StatusCode() int }); ok {
			return sc.StatusCode()
		}
	}
	// Default to 200 OK if no error or no custom status code
	return http.StatusOK
}

type GetEnterpriseResponse struct {
	EnterpriseID string `json:"android_enterprise_id"`
	DefaultResponse
}

type EnterpriseSignupResponse struct {
	Url string `json:"android_enterprise_signup_url"`
	DefaultResponse
}

type EnrollmentTokenResponse struct {
	*EnrollmentToken
	DefaultResponse
}
