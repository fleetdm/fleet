package fleet

import (
	"net/mail"
	"regexp"
	"strings"
)

// GoogleConditionalAccessIntegration holds the app config settings of the Google
// conditional access integration. The service account key is not part of it: it is
// uploaded through the integration endpoint and stored encrypted in mdm_config_assets.
type GoogleConditionalAccessIntegration struct {
	Enabled bool `json:"enabled"`
	// CustomerID is the Google Workspace customer ID with its leading "C", as the Admin console shows it.
	CustomerID string `json:"customer_id"`
	// AdminEmail is the Workspace admin the service account impersonates.
	AdminEmail string `json:"admin_email"`
}

// ConditionalAccessGoogleStatus is the read-only connection state of the Google
// conditional access integration.
type ConditionalAccessGoogleStatus struct {
	ServiceAccountKeyConfigured bool `json:"service_account_key_configured"`
	// State is empty when the integration is off.
	State        ConditionalAccessGoogleState `json:"state"`
	ErrorMessage *string                      `json:"error_message"`
}

// ConditionalAccessGoogleIntegration is the connection state of the Google
// conditional access integration (one global row).
type ConditionalAccessGoogleIntegration struct {
	// CustomerID is the Google customer ID with its leading "C", as the Admin console shows it.
	CustomerID string `db:"customer_id"`
	// AdminEmail is the delegation subject the connection was verified with.
	AdminEmail string `db:"admin_email"`
	// ProxyServerSecret authenticates this server with fleetdm.com and fleetdm.com with this server.
	ProxyServerSecret string `db:"proxy_server_secret" json:"-"`
	// OAuthClientID is Fleet's Web OAuth client ID as returned by fleetdm.com.
	// TODO: the Fleet Android app might not need this client ID; drop it if it doesn't.
	OAuthClientID string                       `db:"oauth_client_id"`
	State         ConditionalAccessGoogleState `db:"state"`
	// ErrorMessage is the reason the integration isn't connected, meant for the admin.
	ErrorMessage *string `db:"error_message"`
}

// ConditionalAccessGoogleState is the connection state of the Google conditional
// access integration.
type ConditionalAccessGoogleState string

const (
	// ConditionalAccessGoogleStatePending is saved before registering with fleetdm.com.
	ConditionalAccessGoogleStatePending            ConditionalAccessGoogleState = "pending"
	ConditionalAccessGoogleStateKeyRequired        ConditionalAccessGoogleState = "key_required"
	ConditionalAccessGoogleStateDelegationFailed   ConditionalAccessGoogleState = "delegation_failed"
	ConditionalAccessGoogleStateRegistrationFailed ConditionalAccessGoogleState = "registration_failed"
	ConditionalAccessGoogleStateConnected          ConditionalAccessGoogleState = "connected"
)

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessGoogleIntegration) AuthzType() string {
	return "conditional_access_google"
}

// Registered reports whether this server is registered with fleetdm.com.
func (c *ConditionalAccessGoogleIntegration) Registered() bool {
	return c != nil && c.ProxyServerSecret != ""
}

// googleCustomerIDRegexp matches a Google customer ID as the Admin console shows it,
// with its leading "C".
var googleCustomerIDRegexp = regexp.MustCompile(`^C[0-9A-Za-z]{6,32}$`)

// ValidateGoogleConditionalAccessSettings trims the customer ID and admin email in
// place and appends any error to invalid, with field names under fieldPrefix. Both are
// required when the integration is enabled.
func ValidateGoogleConditionalAccessSettings(intg *GoogleConditionalAccessIntegration, fieldPrefix string, invalid *InvalidArgumentError) {
	if intg == nil {
		return
	}
	intg.CustomerID = strings.TrimSpace(intg.CustomerID)
	intg.AdminEmail = strings.TrimSpace(intg.AdminEmail)

	if intg.CustomerID == "" {
		if intg.Enabled {
			invalid.Append(fieldPrefix+"customer_id", "customer_id is required")
		}
	} else if !googleCustomerIDRegexp.MatchString(intg.CustomerID) {
		invalid.Append(fieldPrefix+"customer_id", "must be a Google customer ID starting with C, for example C01234abc")
	}

	if intg.AdminEmail == "" {
		if intg.Enabled {
			invalid.Append(fieldPrefix+"admin_email", "admin_email is required")
		}
	} else if addr, err := mail.ParseAddress(intg.AdminEmail); err != nil || addr.Address != intg.AdminEmail {
		invalid.Append(fieldPrefix+"admin_email", "must be a valid email address")
	}
}
