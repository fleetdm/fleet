// Package entratoken gets an Entra ID token carrying the device's Entra device ID.
//
// The authorize request goes through Apple's URL loading system so the Microsoft
// Enterprise SSO extension (Company Portal) can attach the device's Platform SSO
// credentials. The extension only does so for processes whose signing identifier is in
// its AppAllowList, and only in the session of the user that registered the device, so
// the resulting token proves the request came from that device.
package entratoken

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/uuid"
)

// DefaultRedirectURI must be registered on the Entra app under the "iOS / macOS" platform.
// Nothing opens it: the extension returns the authorization code in the response.
const DefaultRedirectURI = "msauth.com.fleetdm.orbit://auth"

const (
	authority = "https://login.microsoftonline.com"
	// No offline_access: the token is only used to prove the device ID, so no refresh token is requested.
	scope = "openid profile"
	// essential makes Entra fail the request instead of omitting the device ID.
	claims = `{"id_token":{"deviceid":{"essential":true}}}`
)

// Request holds the parameters of the token request.
type Request struct {
	TenantID    string
	ClientID    string
	RedirectURI string
	Nonce       string
}

var nonceRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,256}$`)

var errNonceForAnotherHost = errors.New("nonce was not generated for this host")

// nonceIsForHost reports whether Fleet generated the nonce for the host with the given hardware
// UUID; Fleet prefixes each host's nonce with its UUID.
func nonceIsForHost(nonce, hardwareUUID string) bool {
	nonceHost, _, _ := strings.Cut(nonce, ".")
	return hardwareUUID != "" && strings.EqualFold(nonceHost, hardwareUUID)
}

// Validate checks the request parameters.
func (r Request) Validate() error {
	if _, err := uuid.Parse(r.TenantID); err != nil {
		return errors.New("invalid tenant ID")
	}
	if _, err := uuid.Parse(r.ClientID); err != nil {
		return errors.New("invalid client ID")
	}
	if r.RedirectURI == "" {
		return errors.New("missing redirect URI")
	}
	if !nonceRe.MatchString(r.Nonce) {
		return errors.New("invalid nonce")
	}
	return nil
}

// GetIDToken returns the raw ID token. It must run as the user that is logged in to Entra.
func GetIDToken(ctx context.Context, r Request) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	// Refuse other hosts' nonces, so this command can't be used on this device to mint a token
	// that another host then reports as its own.
	hostID, err := hostUUID()
	if err != nil {
		return "", err
	}
	if !nonceIsForHost(r.Nonce, hostID) {
		return "", errNonceForAnotherHost
	}

	q := url.Values{}
	q.Set("client_id", r.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", r.RedirectURI)
	q.Set("scope", scope)
	q.Set("claims", claims)
	q.Set("prompt", "none")
	q.Set("response_mode", "query")
	q.Set("nonce", r.Nonce)
	authorizeURL := fmt.Sprintf("%s/%s/oauth2/v2.0/authorize?%s", authority, r.TenantID, q.Encode())

	// Leave time for redeeming the code if the authorize request is slow.
	authorizeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	resp, err := authorize(authorizeCtx, authorizeURL)
	if err != nil {
		return "", err
	}
	code, err := extractCode(resp, r.RedirectURI)
	if err != nil {
		return "", err
	}
	return redeemCode(ctx, r, code)
}

type authorizeResponse struct {
	// redirectURL is set when the response redirected to the app's redirect URI.
	redirectURL string
	body        []byte
}

var (
	// codeInBodyRe matches the code in the page the extension returns on success.
	codeInBodyRe = regexp.MustCompile(`[?&]code=([^\\&"'<\s]+)`)
	aadErrorRe   = regexp.MustCompile(`AADSTS\d+`)
)

func extractCode(resp *authorizeResponse, redirectURI string) (string, error) {
	if resp.redirectURL != "" {
		u, err := url.Parse(resp.redirectURL)
		if err != nil {
			return "", fmt.Errorf("parse redirect URL: %w", err)
		}
		q := u.Query()
		if code := q.Get("code"); code != "" {
			return code, nil
		}
		if e := q.Get("error"); e != "" {
			return "", fmt.Errorf("authorize: %s: %s", e, q.Get("error_description"))
		}
		return "", errors.New("authorize: redirect without code")
	}
	body := string(resp.body)
	if i := strings.Index(body, redirectURI); i >= 0 {
		if m := codeInBodyRe.FindStringSubmatch(body[i:]); m != nil {
			return url.QueryUnescape(m[1])
		}
	}
	if m := aadErrorRe.FindString(body); m != "" {
		return "", fmt.Errorf("authorize: %s", m)
	}
	return "", errors.New("authorize: no code in response")
}

func redeemCode(ctx context.Context, r Request, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", r.ClientID)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", r.RedirectURI)
	form.Set("scope", scope)
	form.Set("claims", claims)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/oauth2/v2.0/token", authority, r.TenantID), strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second)).Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	var tok struct {
		IDToken          string `json:"id_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(b, &tok); err != nil {
		return "", fmt.Errorf("decode token response (status %d): %w", resp.StatusCode, err)
	}
	if tok.Error != "" {
		return "", fmt.Errorf("token: %s: %s", tok.Error, tok.ErrorDescription)
	}
	if tok.IDToken == "" {
		return "", errors.New("token response without id_token")
	}
	return tok.IDToken, nil
}
