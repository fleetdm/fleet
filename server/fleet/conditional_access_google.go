package fleet

// ConditionalAccessGoogleIntegration holds the connection state of the Google
// conditional access integration. The admin settings (customer ID, admin email,
// service account key) live in Integrations.GoogleConditionalAccess.
type ConditionalAccessGoogleIntegration struct {
	// CustomerID is the Google customer ID registered with fleetdm.com.
	CustomerID string `db:"customer_id"`
	// ProxyServerSecret is the per-server secret fleetdm.com issued for this customer ID.
	ProxyServerSecret string `db:"proxy_server_secret"`
	// OAuthClientID is Fleet's Web OAuth client ID, as returned by fleetdm.com.
	OAuthClientID string `db:"oauth_client_id"`
	// SetupDone is true when the integration is connected.
	SetupDone bool `db:"setup_done"`
	// SetupError holds the last error minting the delegated token, if any.
	SetupError *string `db:"setup_error"`
}

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessGoogleIntegration) AuthzType() string {
	return "conditional_access_google"
}
