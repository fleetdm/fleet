package fleet

import (
	"net/mail"
	"regexp"
	"strings"
)

// GoogleConditionalAccessIntegration is the app config part of the integration. The
// service account key is stored encrypted in mdm_config_assets instead.
type GoogleConditionalAccessIntegration struct {
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

// ConditionalAccessGoogleIntegration is the fleetdm.com registration and connection
// state of the Google conditional access integration (one global row). The customer ID
// and admin email live in the app config only: the registration is always for the
// customer the config names.
type ConditionalAccessGoogleIntegration struct {
	// ProxyServerSecret authenticates this server with fleetdm.com and fleetdm.com with this server.
	ProxyServerSecret string                       `db:"proxy_server_secret" json:"-"`
	State             ConditionalAccessGoogleState `db:"state"`
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
	// ConditionalAccessGoogleStateDeleted marks a removed integration. Its device rows are
	// kept, so adding it back for the same customer finds them; its updated_at is when it
	// was removed.
	ConditionalAccessGoogleStateDeleted ConditionalAccessGoogleState = "deleted"
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
		invalid.Append("customer_id", "customer_id is required")
	} else if !googleCustomerIDRegexp.MatchString(g.CustomerID) {
		invalid.Append("customer_id", "must be a Google customer ID starting with C, for example C01234abc")
	}

	if g.AdminEmail == "" {
		invalid.Append("admin_email", "admin_email is required")
	} else if addr, err := mail.ParseAddress(g.AdminEmail); err != nil || addr.Address != g.AdminEmail {
		invalid.Append("admin_email", "must be a valid email address")
	}
}
