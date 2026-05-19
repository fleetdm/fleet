package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	jwt "github.com/golang-jwt/jwt/v4"
)

// defaultOIDCScopes is the scope string used when PSSOSettings.IdPScopes is
// empty. "openid" is required to get an id_token back, which is where we
// pull the user's claims from.
const defaultOIDCScopes = "openid profile email"

// PSSOOIDCROPGClient validates passwords against any OIDC IdP that exposes
// the OAuth2 Resource Owner Password Grant on its token endpoint. The POC
// has been exercised against Okta first; Entra ID, Auth0, Keycloak, and
// other providers that allow ROPG will work with no code changes, only
// different TokenURL values.
//
// Known limitations:
//   - Okta: ROPG must be explicitly enabled on the application ("Allowed
//     Grant Types → Resource Owner Password" in the Okta app config), and
//     the app type must be Native or Service. SPAs and standard web apps
//     reject the grant.
//   - Entra: MFA-required users fail with conditional access errors;
//     federated (AD FS) users are not supported. Both are upstream
//     constraints, not Fleet bugs.
//   - All providers: ROPG is widely considered deprecated for new
//     deployments. Use a different PSSOIdPClient implementation (e.g. LDAP
//     bind) if your IdP doesn't support it.
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
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// ValidatePasswordAndGetClaims posts the user's credentials to the IdP's
// token endpoint with grant_type=password, then parses the returned
// id_token (JWT) for OIDC-shaped claims.
func (c PSSOOIDCROPGClient) ValidatePasswordAndGetClaims(ctx context.Context, username, password string) (*fleet.PSSOClaims, error) {
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

	client := c.HTTPClient
	if client == nil {
		client = fleethttp.NewClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "post oidc ropg request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read oidc ropg response")
	}

	var parsed oidcTokenResponse
	if jerr := json.Unmarshal(body, &parsed); jerr != nil {
		return nil, ctxerr.Wrap(ctx, jerr, "decode oidc ropg response")
	}
	if resp.StatusCode != http.StatusOK || parsed.Error != "" {
		return nil, fleet.NewAuthFailedError(fmt.Sprintf("idp rejected password: %s %s", parsed.Error, parsed.ErrorDesc))
	}
	if parsed.IDToken == "" {
		return nil, errors.New("idp response missing id_token (is 'openid' in the configured scopes?)")
	}

	return parseOIDCIDTokenClaims(parsed.IDToken)
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
	return out, nil
}

func stringClaim(c jwt.MapClaims, key string) string {
	v, _ := c[key].(string)
	return v
}
