package godep

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type AccountDrivenEnrollmentProfileResponse struct {
	MDMServiceDiscoveryURL string    `json:"mdm_service_discovery_url"`
	LastUpdatedTimestamp   time.Time `json:"last_updated_timestamp"`
}

// FetchAccountDrivenEnrollmentServiceDiscovery uses the Apple "Fetch Account Driven Enrollment Service Discovery" API endpoint
// to get the service discovery URL and last updated time for account-driven enrollment.
// https://developer.apple.com/documentation/devicemanagement/fetch-account-driven-enrollment-profile
func (c *Client) FetchAccountDrivenEnrollmentServiceDiscovery(ctx context.Context, name string) (*AccountDrivenEnrollmentProfileResponse, error) {
	resp := &AccountDrivenEnrollmentProfileResponse{}
	if err := c.doWithAfterHook(ctx, name, http.MethodGet, "/account-driven-enrollment/profile", nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AssignAccountDrivenEnrollmentServiceDiscovery uses the Apple "Assign Account Driven Enrollment Service Discovery" API endpoint
// to assign the service discovery URL for account-driven enrollment.
// https://developer.apple.com/documentation/devicemanagement/fetch-account-driven-enrollment-profile
func (c *Client) AssignAccountDrivenEnrollmentServiceDiscovery(ctx context.Context, name string, url string) error {
	resp := &json.RawMessage{}
	return c.doWithAfterHook(ctx, name, http.MethodPost, "/account-driven-enrollment/profile",
		map[string]string{"mdm_service_discovery_url": url}, resp)
}

// TODO: Implement RemoveAccountDrivenEnrollmentProfile https://developer.apple.com/documentation/devicemanagement/remove-account-driven-enrollment-profile

// TODO: Implement error checks to cover https://developer.apple.com/documentation/devicemanagement/assign-account-driven-enrollment-profile#response-codes

// IsServiceDiscoveryNotFound indicates that Apple can’t find the MDM Service Discovery URL.
// https://developer.apple.com/documentation/devicemanagement/fetch-account-driven-enrollment-profile#response-codes
func IsServiceDiscoveryNotFound(err error) bool {
	return httpErrorContains(err, http.StatusNotFound, "NOT_FOUND")
}

// IsServiceDiscoveryNotSupported indicates that the organization is not supported for account-driven enrollment according to Apple.
// https://developer.apple.com/documentation/devicemanagement/fetch-account-driven-enrollment-profile#response-codes
func IsServiceDiscoveryNotSupported(err error) bool {
	return httpErrorContains(err, http.StatusBadRequest, "ORG_NOT_SUPPORTED")
}
