package godep

import (
	"context"
	"net/http"
)

// AccountResponse corresponds to the Apple DEP API "AccountDetail" structure.
// See https://developer.apple.com/documentation/devicemanagement/accountdetail
type AccountResponse struct {
	AdminID       string `json:"admin_id"`
	FacilitatorID string `json:"facilitator_id"`
	OrgAddress    string `json:"org_address"`
	OrgEmail      string `json:"org_email"`
	OrgID         string `json:"org_id"`
	OrgIDHash     string `json:"org_id_hash"`
	OrgName       string `json:"org_name"`
	OrgPhone      string `json:"org_phone"`
	OrgType       string `json:"org_type"`
	OrgVersion    string `json:"org_version"`
	ServerName    string `json:"server_name"`
	ServerUUID    string `json:"server_uuid"`
	URLs          []URL  `json:"urls"`
}

// URL corresponds to the Apple DEP API "Url" structure.
// See https://developer.apple.com/documentation/devicemanagement/url
type URL struct {
	HTTPMethod []string `json:"http_method"`
	Limit      *Limit   `json:"limit"`
	URI        string   `json:"uri"`
}

// Limit corresponds to the Apple DEP API "Limit" structure.
// See https://developer.apple.com/documentation/devicemanagement/limit
type Limit struct {
	Default int `json:"default"`
	Maximum int `json:"maximum"`
}

// AccountDetail uses the Apple "Get Account Detail" API endpoint to get the
// account details for the current DEP authentication token.
// See https://developer.apple.com/documentation/devicemanagement/get_account_detail
func (c *Client) AccountDetail(ctx context.Context, name string) (*AccountResponse, error) {
	resp := new(AccountResponse)
	return resp, c.doWithAfterHook(ctx, name, http.MethodGet, "/account", nil, resp)
}

// IsTermsNotSigned returns true if err is a DEP "terms and conditions not
// signed" error. Per Apple this indicates the Terms and Conditions must be
// accepted by the user.
// See https://developer.apple.com/documentation/devicemanagement/device_assignment/authenticating_with_a_device_enrollment_program_dep_server/interpreting_error_codes
func IsTermsNotSigned(err error) bool {
	return httpErrorContains(err, http.StatusForbidden, "T_C_NOT_SIGNED") ||
		authErrorContains(err, http.StatusForbidden, "T_C_NOT_SIGNED")
}
