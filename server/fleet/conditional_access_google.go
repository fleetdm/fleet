package fleet

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
