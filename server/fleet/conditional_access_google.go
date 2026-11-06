package fleet

import (
	"net/mail"
	"regexp"
	"strings"
)

// GoogleConditionalAccessIntegration is the app config part of the integration. The
// service account key is stored encrypted in mdm_config_assets instead.
type GoogleConditionalAccessIntegration struct {
	Enabled bool `json:"enabled"`
	// CustomerID includes the leading "C".
	CustomerID string `json:"customer_id"`
	// AdminEmail is the Workspace admin the service account impersonates.
	AdminEmail string `json:"admin_email"`
}

// ConditionalAccessGoogleStatus is the integration's connection state.
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

var googleCustomerIDRegexp = regexp.MustCompile(`^C[0-9A-Za-z]{6,32}$`)

// Validate trims the fields in place and appends any error to invalid.
func (g *GoogleConditionalAccessIntegration) Validate(invalid *InvalidArgumentError) {
	g.CustomerID = strings.TrimSpace(g.CustomerID)
	g.AdminEmail = strings.TrimSpace(g.AdminEmail)

	if g.CustomerID == "" {
		if g.Enabled {
			invalid.Append("customer_id", "customer_id is required")
		}
	} else if !googleCustomerIDRegexp.MatchString(g.CustomerID) {
		invalid.Append("customer_id", "must be a Google customer ID starting with C, for example C01234abc")
	}

	if g.AdminEmail == "" {
		if g.Enabled {
			invalid.Append("admin_email", "admin_email is required")
		}
	} else if addr, err := mail.ParseAddress(g.AdminEmail); err != nil || addr.Address != g.AdminEmail {
		invalid.Append("admin_email", "must be a valid email address")
	}
}
