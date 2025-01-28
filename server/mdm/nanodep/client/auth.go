package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gomodule/oauth1/oauth"
)

// sessionContentType is the exact header required by the `depsim` DEP
// simulator for the /session endpoint.
const sessionContentType = "application/json;charset=UTF8"

// ErrEmptyAuthSessionToken occurs with a valid JSON session response but
// contains an empty session token.
var ErrEmptyAuthSessionToken = errors.New("empty auth session token")

// AuthError encapsulates an HTTP response error from the /session endpoint.
// The API returns error information in the request body.
type AuthError struct {
	Body       []byte
	Status     string
	StatusCode int
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("DEP auth error: %s: %s", e.Status, string(e.Body))
}

// NewAuthError creates and returns a new AuthError from r. Note this reads
// r.Body and you are responsible for Closing it.
func NewAuthError(r *http.Response) error {
	body, readErr := io.ReadAll(r.Body)
	err := &AuthError{
		Body:       body,
		Status:     r.Status,
		StatusCode: r.StatusCode,
	}
	if readErr != nil {
		return fmt.Errorf("reading body of DEP auth error: %v: %w", err, readErr)
	}
	return err
}

// OAuth1Tokens represents the token Apple DEP OAuth1 authentication tokens.
type OAuth1Tokens struct {
	ConsumerKey       string    `json:"consumer_key"`
	ConsumerSecret    string    `json:"consumer_secret"`
	AccessToken       string    `json:"access_token"`
	AccessSecret      string    `json:"access_secret"`
	AccessTokenExpiry time.Time `json:"access_token_expiry"`
}

// Valid performs sanity checks to make sure t appears to be valid DEP server OAuth 1 tokens.
func (t *OAuth1Tokens) Valid() bool {
	if t == nil {
		return false
	}
	if t.ConsumerKey != "" && t.ConsumerSecret != "" && t.AccessToken != "" && t.AccessSecret != "" {
		return true
	}
	return false
}

// SetAuthorizationHeader sets the OAuth1 Authorization HTTP request header
// using the supplied DEP tokens. Intended for the DEP /session endpoint.
// See https://developer.apple.com/documentation/devicemanagement/device_assignment/authenticating_with_a_device_enrollment_program_dep_server
func SetAuthorizationHeader(tokens *OAuth1Tokens, req *http.Request) error {
	consumerCreds := oauth.Credentials{
		Token:  tokens.ConsumerKey,
		Secret: tokens.ConsumerSecret,
	}
	oauthClient := oauth.Client{
		SignatureMethod: oauth.HMACSHA1, // HMAC-SHA1 is required by Apple
		TokenRequestURI: req.URL.String(),
		Credentials:     consumerCreds,
	}
	accessCreds := oauth.Credentials{
		Token:  tokens.AccessToken,
		Secret: tokens.AccessSecret,
	}
	return oauthClient.SetAuthorizationHeader(
		req.Header,
		&accessCreds,
		req.Method,
		req.URL,
		nil,
	)
}

// DoAuth performs OAuth1 authentication to the Apple DEP server and returns
// the 'auth_session_token' from the JSON response.
func DoAuth(client Doer, req *http.Request, tokens *OAuth1Tokens) (string, error) {
	err := SetAuthorizationHeader(tokens, req)
	if err != nil {
		return "", err
	}
	if _, ok := req.Header["Content-Type"]; !ok {
		// required for the simulator
		req.Header.Set("Content-Type", sessionContentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", NewAuthError(resp)
	}
	var authSessionToken struct {
		AuthSessionToken string `json:"auth_session_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&authSessionToken)
	if err == nil && authSessionToken.AuthSessionToken == "" {
		err = ErrEmptyAuthSessionToken
	}
	return authSessionToken.AuthSessionToken, err
}
