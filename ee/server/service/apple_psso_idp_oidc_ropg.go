package service

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	jwt "github.com/golang-jwt/jwt/v4"
)

// defaultOIDCScopes is the scope string used when PSSOSettings.IdPScopes is
// empty. "openid" is required to get an id_token back, which is where we
// pull the user's claims from.
const defaultOIDCScopes = "openid profile email"

// maxOIDCTokenResponseSize bounds how much of the IdP token response Fleet
// will read. Real responses (id_token + refresh_token JSON) are a few KB.
const maxOIDCTokenResponseSize = 1 << 20 // 1 MiB

// oauthErrInvalidGrant is the OAuth 2.0 token error (RFC 6749 section 5.2) an
// IdP returns when the resource owner's credentials are wrong.
const oauthErrInvalidGrant = "invalid_grant"

// oauthErrorFields lists, in order, the JSON fields an IdP's token error code
// and description are read from: RFC 6749's error/error_description, then
// Okta's own format, which it uses when the client_id is unknown. Add a pair
// here for a provider that reports errors elsewhere.
var oauthErrorFields = []struct{ code, description string }{
	{"error", "error_description"},
	{"errorCode", "errorSummary"},
}

// ropgVerifyUsername is the account VerifyConfig signs in as. It can't exist
// (the .invalid TLD is reserved), and it reads as a configuration check in the
// customer's IdP sign-in logs rather than a guessed password.
const ropgVerifyUsername = "fleet-config-check@invalid.invalid"

// ropgVerifyTimeout bounds VerifyConfig so a slow IdP can't stall a settings
// save. Sign-in has no such limit. A var so a test can shorten it; tests that
// change it must not run in parallel.
var ropgVerifyTimeout = 10 * time.Second

// ropgVerifyOutcome is what an IdP's error code says about Fleet's settings
// when VerifyConfig signs in as a user that doesn't exist.
type ropgVerifyOutcome int

const (
	ropgVerifyUnknown ropgVerifyOutcome = iota
	// ropgVerifyOK: the IdP accepted the client and rejected only the user,
	// which it checks after authenticating the client (RFC 6749 section 5.2).
	ropgVerifyOK
	ropgVerifyClientCredentials
	ropgVerifyAppSetup
	// ropgVerifyTokenURL: the IdP answered with a non-OAuth API error, so the
	// URL isn't its token endpoint.
	ropgVerifyTokenURL
)

// ropgVerifyOutcomes maps token error codes to what they mean for the settings.
// Any code not listed is reported to the admin with the IdP's own description.
// Add a row for a provider-specific code.
var ropgVerifyOutcomes = map[string]ropgVerifyOutcome{
	oauthErrInvalidGrant:     ropgVerifyOK,
	"invalid_client":         ropgVerifyClientCredentials,
	"unauthorized_client":    ropgVerifyAppSetup,
	"unsupported_grant_type": ropgVerifyAppSetup,
	"invalid_scope":          ropgVerifyAppSetup,
	"access_denied":          ropgVerifyAppSetup,
	// Okta: the authorization server's access policy doesn't allow the
	// Resource Owner Password grant for this app.
	"no_matching_policy":          ropgVerifyAppSetup,
	"password_auth_denied_policy": ropgVerifyAppSetup,
	// Okta API errors for a URL that isn't an OAuth endpoint (wrong method, not found).
	"E0000022": ropgVerifyTokenURL,
	"E0000007": ropgVerifyTokenURL,
}

// ropgDecodeError is a token endpoint reply that isn't a JSON OAuth response.
type ropgDecodeError struct {
	status int
	err    error
}

func (e *ropgDecodeError) Error() string { return fmt.Sprintf("HTTP %d: %v", e.status, e.err) }
func (e *ropgDecodeError) Unwrap() error { return e.err }

// pssoIdPClientConfigError is an IdP token error caused by Fleet's IdP client
// settings rather than the user. The IdP's details are logged via Internal but
// never sent to the device: a plain error's text would be.
type pssoIdPClientConfigError struct {
	detail string
}

func (e *pssoIdPClientConfigError) Error() string {
	return "identity provider rejected Fleet's client configuration"
}

func (e *pssoIdPClientConfigError) Internal() string { return e.detail }

// StatusCode is a 4xx because a customer's misconfiguration would otherwise
// return a 5xx on every sign-in until fixed, setting off server error alerts.
// Any status but 401 keeps the device from reporting a bad password.
func (e *pssoIdPClientConfigError) StatusCode() int { return http.StatusBadRequest }

// PSSOOIDCROPGClient validates passwords against any OIDC IdP that exposes
// the OAuth2 Resource Owner Password Grant on its token endpoint
type PSSOOIDCROPGClient struct {
	// TokenURL is the full URL of the IdP's token endpoint.
	TokenURL     string
	ClientID     string
	ClientSecret string
	// Scopes is the space-separated scope string. Empty falls back to
	// defaultOIDCScopes.
	Scopes string

	// HTTPClient may be overridden in tests. nil falls back to fleethttp.
	HTTPClient *http.Client
}

// oidcTokenResponse models the subset of fields we use from a standard
// OIDC token endpoint response.
type oidcTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// ropgTokenReply is an IdP token endpoint response.
type ropgTokenReply struct {
	status      int
	token       oidcTokenResponse
	code        string
	description string
}

// postTokenRequest sends a Resource Owner Password grant request to the IdP
// token endpoint and decodes its JSON reply. A transport failure or a reply
// that isn't JSON is returned as an error.
func (c PSSOOIDCROPGClient) postTokenRequest(ctx context.Context, client *http.Client, username, password string) (*ropgTokenReply, error) {
	if c.TokenURL == "" || c.ClientID == "" || c.ClientSecret == "" {
		return nil, errors.New("oidc ropg client is missing token_url, client_id, or client_secret")
	}

	scopes := c.Scopes
	if scopes == "" {
		scopes = defaultOIDCScopes
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("username", username)
	form.Set("password", password)
	form.Set("scope", scopes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build oidc ropg request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "post oidc ropg request")
	}
	defer resp.Body.Close()

	// Cap the read so a misbehaving (or misconfigured) IdP can't exhaust
	// memory; a token response is a few KB.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOIDCTokenResponseSize))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read oidc ropg response")
	}

	reply := &ropgTokenReply{status: resp.StatusCode}
	if err := json.Unmarshal(body, &reply.token); err != nil {
		return nil, ctxerr.Wrap(ctx, &ropgDecodeError{status: resp.StatusCode, err: err}, "decode oidc ropg response")
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(body, &fields) // can't fail: the body just decoded as an object
	for _, f := range oauthErrorFields {
		var code, description string
		if raw, ok := fields[f.code]; ok {
			if err := json.Unmarshal(raw, &code); err != nil {
				return nil, ctxerr.Wrap(ctx, &ropgDecodeError{status: resp.StatusCode, err: err}, "decode oidc ropg response "+f.code)
			}
		}
		if raw, ok := fields[f.description]; ok {
			if err := json.Unmarshal(raw, &description); err != nil {
				return nil, ctxerr.Wrap(ctx, &ropgDecodeError{status: resp.StatusCode, err: err}, "decode oidc ropg response "+f.description)
			}
		}
		if code != "" {
			reply.code, reply.description = code, description
			break
		}
	}
	return reply, nil
}

// ValidatePasswordAndGetClaims posts the user's credentials to the IdP's
// token endpoint with grant_type=password, then parses the returned
// id_token (JWT) for OIDC-shaped claims.
func (c PSSOOIDCROPGClient) ValidatePasswordAndGetClaims(ctx context.Context, username, password string) (*fleet.PSSOClaims, error) {
	client := c.HTTPClient
	if client == nil {
		client = fleethttp.NewClient(fleethttp.WithNoTimeout())
	}
	reply, err := c.postTokenRequest(ctx, client, username, password)
	if err != nil {
		return nil, err
	}
	if reply.status != http.StatusOK || reply.code != "" {
		// invalid_grant is the IdP rejecting the user's credentials, the only
		// case the user can fix. Any other error (invalid_client,
		// unauthorized_client, ...) means Fleet's IdP client settings are wrong,
		// which the device must not present as a bad password.
		if reply.code == oauthErrInvalidGrant {
			return nil, fleet.NewAuthFailedError(fmt.Sprintf("idp rejected password: %s %s", reply.code, reply.description))
		}
		return nil, &pssoIdPClientConfigError{detail: fmt.Sprintf(
			"idp token endpoint returned HTTP %d %q %q: check the Apple account provisioning token URL, client ID, and client secret",
			reply.status, reply.code, reply.description)}
	}
	parsed := reply.token
	if parsed.IDToken == "" {
		return nil, errors.New("idp response missing id_token (is 'openid' in the configured scopes?)")
	}

	claims, err := parseOIDCIDTokenClaims(parsed.IDToken)
	if err != nil {
		return nil, err
	}
	claims.RefreshToken = parsed.RefreshToken
	claims.ExpiresIn = parsed.ExpiresIn
	return claims, nil
}

// VerifyConfig checks the token URL, client ID, and client secret by signing
// in as ropgVerifyUsername with a random password. The IdP only rejects that
// user (invalid_grant) once it has authenticated Fleet's client, so any other
// answer means the settings are wrong, reported as an *fleet.InvalidArgumentError.
func (c PSSOOIDCROPGClient) VerifyConfig(ctx context.Context) error {
	const (
		urlField     = "mdm.apple_account_provisioning.oauth_idp_token_url"
		settingField = "mdm.apple_account_provisioning"
	)

	ctx, cancel := context.WithTimeout(ctx, ropgVerifyTimeout)
	defer cancel()
	client := c.HTTPClient
	if client == nil {
		client = fleethttp.NewClient(fleethttp.WithTimeout(ropgVerifyTimeout))
	}

	password := make([]byte, 16)
	if _, err := rand.Read(password); err != nil {
		return ctxerr.Wrap(ctx, err, "generate idp verification password")
	}
	reply, err := c.postTokenRequest(ctx, client, ropgVerifyUsername, hex.EncodeToString(password))
	if err != nil {
		// The raw error can name internal hosts or resolvers, so it only goes to
		// the server log; the admin gets a plain description. A 4xx is otherwise
		// logged at debug level, where nobody troubleshooting would see it.
		logging.WithExtras(ctx, "idp_verify_error", err.Error())
		logging.WithLevel(ctx, slog.LevelInfo)
		return fleet.NewInvalidArgumentError(urlField, describeTokenURLFailure(c.TokenURL, err))
	}

	// The IdP's text is shown to the admin; cap it so an odd response can't
	// swamp the error.
	idpDetail := reply.code
	if reply.description != "" {
		idpDetail += ": " + reply.description
	}
	if len(idpDetail) > 300 {
		idpDetail = idpDetail[:300] + "…"
	}
	switch {
	case reply.code == "" && reply.status == http.StatusOK:
		return fleet.NewInvalidArgumentError(urlField, fmt.Sprintf(
			"The token URL returned a token for %s, a test user that shouldn't exist. Check that it points at your IdP's token endpoint.",
			ropgVerifyUsername))
	case reply.code == "":
		return fleet.NewInvalidArgumentError(urlField, fmt.Sprintf(
			"The token URL didn't return an OAuth error (HTTP %d). Check that it points at your IdP's token endpoint.", reply.status))
	}
	switch ropgVerifyOutcomes[reply.code] {
	case ropgVerifyOK:
		return nil
	case ropgVerifyClientCredentials:
		return fleet.NewInvalidArgumentError(settingField, fmt.Sprintf(
			"Your IdP rejected the client ID or client secret (%s).", idpDetail))
	case ropgVerifyTokenURL:
		return fleet.NewInvalidArgumentError(urlField, fmt.Sprintf(
			"The token URL isn't your IdP's OAuth token endpoint (%s).", idpDetail))
	case ropgVerifyAppSetup:
		return fleet.NewInvalidArgumentError(settingField, fmt.Sprintf(
			"Your IdP didn't let this app sign users in with a password (%s). Check that the app exists, allows the OAuth Resource Owner Password grant, and is permitted by your IdP's access policies.", idpDetail))
	default:
		return fleet.NewInvalidArgumentError(settingField, fmt.Sprintf(
			"Your IdP rejected Fleet's test sign-in (%s).", idpDetail))
	}
}

// describeTokenURLFailure explains why VerifyConfig got no OAuth reply from
// the token URL, in terms an admin can act on.
func describeTokenURLFailure(tokenURL string, err error) string {
	host := tokenURL
	if u, perr := url.Parse(tokenURL); perr == nil && u.Host != "" {
		host = u.Host
	}
	if decodeErr, ok := errors.AsType[*ropgDecodeError](err); ok {
		return fmt.Sprintf("The token URL didn't return an OAuth response (HTTP %d). Check that it points at your IdP's token endpoint.", decodeErr.status)
	}
	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr.IsNotFound {
		return fmt.Sprintf("Fleet couldn't find the host %s. Check the token URL.", host)
	}
	if netErr, ok := errors.AsType[net.Error](err); (ok && netErr.Timeout()) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("The token URL didn't respond within %s.", ropgVerifyTimeout)
	}
	if _, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		return fmt.Sprintf("Fleet couldn't verify the TLS certificate of %s.", host)
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Sprintf("Fleet couldn't connect to %s.", host)
	}
	return "Fleet couldn't reach the token URL."
}

// parseOIDCIDTokenClaims decodes the id_token JWT body without verifying
// its signature. Verification isn't necessary here because we just
// received the token directly from the issuer over a TLS-protected
// channel — the JWT is effectively a structured response, not a
// cross-trust assertion. (If we ever start accepting id_tokens from
// elsewhere, this needs to change.)
func parseOIDCIDTokenClaims(idToken string) (*fleet.PSSOClaims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}
	if _, _, err := parser.ParseUnverified(idToken, claims); err != nil {
		return nil, fmt.Errorf("parse id_token: %w", err)
	}
	out := &fleet.PSSOClaims{
		Subject:           stringClaim(claims, "sub"),
		Email:             stringClaim(claims, "email"),
		Name:              stringClaim(claims, "name"),
		PreferredUsername: stringClaim(claims, "preferred_username"),
	}
	if out.Subject == "" {
		return nil, errors.New("id_token missing sub claim")
	}
	// Carry every other claim through in Extra so the PSSO login handler can
	// forward the ones it allows into the minted id_token. Skip the registered
	// claims Fleet sets itself when re-signing and the typed claims captured
	// above, so Extra holds only the IdP's custom claims.
	for k, v := range claims {
		if _, reserved := oidcCapturedClaims[k]; reserved {
			continue
		}
		if out.Extra == nil {
			out.Extra = make(map[string]any, len(claims))
		}
		out.Extra[k] = v
	}
	return out, nil
}

// oidcCapturedClaims are id_token claims excluded from Extra so it carries only
// the IdP's custom claims: the ones parseOIDCIDTokenClaims promotes to a typed
// PSSOClaims field, the registered claims Fleet overwrites when re-signing, and
// the standard OIDC session claims. None of these start with the account
// prefix today, so excluding them is defense-in-depth against the forwarding
// rule changing later.
var oidcCapturedClaims = map[string]struct{}{
	"sub": {}, "email": {}, "name": {}, "preferred_username": {},
	"iss": {}, "aud": {}, "exp": {}, "iat": {}, "nbf": {}, "jti": {}, "nonce": {},
	"auth_time": {}, "acr": {}, "amr": {}, "azp": {}, "at_hash": {}, "c_hash": {}, "sid": {},
}

func stringClaim(c jwt.MapClaims, key string) string {
	v, _ := c[key].(string)
	return v
}
