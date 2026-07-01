package fleet

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
)

// PSSODevice marks a Mac host as Apple Platform SSO-registered. It carries no
// key material itself. The device's public keys live in PSSOKey rows
type PSSODevice struct {
	HostUUID  string    `db:"host_uuid"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// PSSOKeyType discriminates a device's signing key from its encryption key.
type PSSOKeyType string

const (
	PSSOKeyTypeSigning    PSSOKeyType = "signing"
	PSSOKeyTypeEncryption PSSOKeyType = "encryption"
)

// PSSOKey is one of a registered device's public keys, indexed by kid (base64
// SHA-256 of the key bytes) so the server can resolve the owning device when
// an extension presents a JWT with that kid in its header. A host may hold
// several keys of the same type: re-registration adds new keys without
// invalidating old ones.
type PSSOKey struct {
	KID       string      `db:"kid"`
	HostUUID  string      `db:"host_uuid"`
	KeyType   PSSOKeyType `db:"key_type"`
	PEM       string      `db:"pem"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
}

// PSSOClaims is the OIDC-shaped claim set the upstream IdP returns after a
// successful password validation. These are embedded in the PSSO response JWE
// sent back to the Mac extension.
type PSSOClaims struct {
	Subject           string         `json:"sub"`
	Email             string         `json:"email,omitempty"`
	Name              string         `json:"name,omitempty"`
	PreferredUsername string         `json:"preferred_username,omitempty"`
	Extra             map[string]any `json:"extra,omitempty"`

	// RefreshToken and ExpiresIn carry the upstream IdP's OAuth token-response
	// fields through to the PSSO login response Fleet returns to the device.
	// The device treats the refresh token as opaque (used for silent SSO
	// renewal); ExpiresIn is the access/refresh token lifetime in seconds.
	RefreshToken string `json:"-"`
	ExpiresIn    int    `json:"-"`
}

// PSSOIdPClient validates a username/password pair against the upstream IdP
// and returns OIDC-shaped claims on success. The shipped implementation is a
// generic OIDC ROPG client (Okta-first, also tested against Entra and other
// providers).
type PSSOIdPClient interface {
	ValidatePasswordAndGetClaims(ctx context.Context, username, password string) (*PSSOClaims, error)
}

// PSSONonceStore is a short-lived store for the nonces issued by the PSSO
// nonce endpoint and consumed (single-use) on every token request. The Redis
// implementation lives in server/mdm/psso/internal/redis_nonces_store.
type PSSONonceStore interface {
	Store(ctx context.Context, nonce string, ttl time.Duration) error
	Consume(ctx context.Context, nonce string) (ok bool, err error)
}

// PSSODeviceRegistrationRequest carries the device-key enrollment the Mac
// extension POSTs to the PSSO registration endpoint. In Password mode this is
// a pure key registration: the extension generates Secure Enclave signing +
// encryption keypairs and submits the public halves plus their kids. User
// identity is established later, at each password login on the token endpoint.
//
// RegistrationToken is the Fleet-signed JWT delivered to the extension via the
// configuration profile's RegistrationToken key; it authenticates the device
// and binds the registration to a specific host (its subject). The host UUID is
// derived from the token.
type PSSODeviceRegistrationRequest struct {
	DeviceUUID          string `json:"device_uuid"`
	DeviceSigningKey    string `json:"device_signing_key"`
	DeviceEncryptionKey string `json:"device_encryption_key"`
	SigningKeyID        string `json:"signing_key_id"`
	EncryptionKeyID     string `json:"encryption_key_id"`
	RegistrationToken   string `json:"registration_token"`
}

// AppleAccountProvisioning is the macOS local account provisioning / Platform
// SSO password sync configuration stored on AppConfig.MDM. The IdP fields are
// generic OAuth2 ROPG credentials (the oauth_ prefix leaves room for other
// auth methods, e.g. LDAP, later).
//
// The client secret is never persisted in the AppConfig JSON: on write it's
// stripped out and stored encrypted in mdm_config_assets, and the API only
// returns the masked value. token URL + client ID are stored in the JSON.
type AppleAccountProvisioning struct {
	// OAuthIdPTokenURL is the upstream OIDC token endpoint used for the ROPG
	// (grant_type=password) flow at sign-in.
	// Okta example:  https://dev-12345.okta.com/oauth2/default/v1/token
	// Entra example: https://login.microsoftonline.com/<tenant>/oauth2/v2.0/token
	OAuthIdPTokenURL optjson.String `json:"oauth_idp_token_url"`
	// OAuthIdPClientID is the client/application ID registered with the upstream IdP.
	OAuthIdPClientID optjson.String `json:"oauth_idp_client_id"`
	// OAuthIdPClientSecret is the client secret registered with the upstream IdP.
	// Stored in mdm_config_assets, not here; this field carries the masked value
	// in API responses and the caller-supplied value on writes.
	OAuthIdPClientSecret optjson.String `json:"oauth_idp_client_secret"`
}

// Configured reports whether the public IdP fields required to operate the
// feature are present. The client secret lives in mdm_config_assets and is not
// part of this check; the write path guarantees a stored secret whenever these
// are set.
func (a AppleAccountProvisioning) Configured() bool {
	return a.OAuthIdPTokenURL.Value != "" && a.OAuthIdPClientID.Value != ""
}

// PSSOSettings is the resolved Platform SSO configuration the service flows
// operate on. It is assembled per request from AppConfig (public IdP fields
// plus the Fleet server URL) and mdm_config_assets (the client secret); it is
// not stored or serialized on its own.
type PSSOSettings struct {
	// IssuerURL is Fleet's own base URL (server_settings.server_url), used as the
	// token issuer and to build the AASA/JWKS URLs.
	IssuerURL string
	// IdPTokenURL is the upstream OIDC token endpoint used for the
	// ROPG (grant_type=password) flow at sign-in.
	IdPTokenURL string
	// IdPClientID is the client/application ID registered with the upstream IdP.
	IdPClientID string
	// IdPClientSecret is the client secret registered with the upstream IdP,
	// loaded from mdm_config_assets.
	IdPClientSecret string
	// IdPScopes is the space-separated scope string sent on both the
	// authorize and token requests. Defaults to "openid profile email" when
	// empty.
	IdPScopes string
}
