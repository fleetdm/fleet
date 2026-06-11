package fleet

import (
	"context"
	"time"
)

// PSSODevice marks a Mac host as Apple Platform SSO-registered. It carries no
// key material itself — the device's public keys live in PSSOKey rows that
// cascade-delete with this record, so removing it completely clears a host's
// PSSO registration. HostUUID is the hardware UUID (matches hosts.uuid; no FK,
// like other host-adjacent tables).
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
// providers); a deterministic test stub is also provided.
type PSSOIdPClient interface {
	ValidatePasswordAndGetClaims(ctx context.Context, username, password string) (*PSSOClaims, error)
}

// PSSONonceStore is a short-lived store for the nonces issued by the PSSO
// /nonce endpoint and consumed in registration and token flows. The Redis
// implementation lives in server/mdm/psso/internal/redis_nonces_store.
type PSSONonceStore interface {
	Store(ctx context.Context, nonce string, ttl time.Duration) error
	Consume(ctx context.Context, nonce string) (ok bool, err error)
}

// PSSORegisterRequest carries the device-key enrollment the Mac extension
// POSTs to /mdm/apple/psso/register. In Password mode this is a pure key
// registration: the extension generates Secure Enclave signing + encryption
// keypairs and submits the public halves plus their kids and the hardware
// device UUID. There is no OAuth code/state — user identity is established
// later at each password login. Field names match the form keys the extension
// constructs (signPubKey/encPubKey carry the PEM-encoded public keys).
type PSSORegisterRequest struct {
	DeviceUUID          string `json:"deviceUUID"           form:"deviceUUID"`
	DeviceSigningKey    string `json:"deviceSigningKey"     form:"signPubKey"`
	DeviceEncryptionKey string `json:"deviceEncryptionKey"  form:"encPubKey"`
	SignKeyID           string `json:"signKeyID"            form:"signKeyID"`
	EncKeyID            string `json:"encKeyID"             form:"encKeyID"`
}

// PSSOSettings holds the global Apple Platform SSO configuration: which
// extension to bind to, what upstream OIDC IdP to proxy password validation
// to, and Fleet's own issuer URL.
//
// IdP-side fields are generic OAuth2/OIDC — Fleet just needs the authorize
// and token URLs plus client credentials. The POC has been exercised against
// Okta first; Entra ID, Google, and any other OIDC provider that exposes
// ROPG (grant_type=password) on its token endpoint should work with no code
// changes, only different URLs.
//
// TODO: IdPClientSecret needs masking on API responses before this leaves
// the POC stage — model the existing AppConfig secret-masking pattern.
type PSSOSettings struct {
	// Enabled toggles the PSSO endpoints on/off at the service layer.
	Enabled bool `json:"enabled"`
	// IssuerURL is the Fleet base URL the extension talks to (e.g. https://fleet.example.com).
	IssuerURL string `json:"issuer_url"`
	// IdPAuthorizeURL is the upstream OIDC authorize endpoint used during
	// device registration (browser auth code flow).
	// Okta example:  https://dev-12345.okta.com/oauth2/default/v1/authorize
	// Entra example: https://login.microsoftonline.com/<tenant>/oauth2/v2.0/authorize
	IdPAuthorizeURL string `json:"idp_authorize_url"`
	// IdPTokenURL is the upstream OIDC token endpoint used for the
	// ROPG (grant_type=password) flow at sign-in.
	// Okta example:  https://dev-12345.okta.com/oauth2/default/v1/token
	// Entra example: https://login.microsoftonline.com/<tenant>/oauth2/v2.0/token
	IdPTokenURL string `json:"idp_token_url"`
	// IdPClientID is the client/application ID registered with the upstream IdP.
	IdPClientID string `json:"idp_client_id"`
	// IdPClientSecret is the client secret registered with the upstream IdP.
	IdPClientSecret string `json:"idp_client_secret"`
	// IdPScopes is the space-separated scope string sent on both the
	// authorize and token requests. Defaults to "openid profile email" when
	// empty.
	IdPScopes string `json:"idp_scopes"`
}
