package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestPSSOOIDCROPGClientIdPErrors(t *testing.T) {
	newClient := func(t *testing.T, status int, body string) PSSOOIDCROPGClient {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return PSSOOIDCROPGClient{
			TokenURL:     srv.URL,
			ClientID:     "client",
			ClientSecret: "secret",
			HTTPClient:   srv.Client(),
		}
	}
	isAuthFailed := func(err error) bool {
		var authErr *fleet.AuthFailedError
		return errors.As(err, &authErr)
	}

	t.Run("wrong password is an auth failure", func(t *testing.T) {
		c := newClient(t, http.StatusBadRequest,
			`{"error":"invalid_grant","error_description":"The credentials provided were invalid."}`)
		_, err := c.ValidatePasswordAndGetClaims(t.Context(), "user", "wrong")
		require.Error(t, err)
		require.True(t, isAuthFailed(err), "got %v", err)
	})

	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"wrong client secret", http.StatusUnauthorized,
			`{"error":"invalid_client","error_description":"The client secret supplied for a confidential client is invalid."}`},
		{"client not allowed the password grant", http.StatusBadRequest,
			`{"error":"unauthorized_client","error_description":"The client is not authorized to use the provided grant type."}`},
		{"password grant disabled", http.StatusBadRequest,
			`{"error":"unsupported_grant_type","error_description":"The authorization grant type is not supported."}`},
		{"non-200 without an OAuth error", http.StatusInternalServerError, `{}`},
	} {
		t.Run(tc.name+" is a config error, not an auth failure", func(t *testing.T) {
			c := newClient(t, tc.status, tc.body)
			_, err := c.ValidatePasswordAndGetClaims(t.Context(), "user", "correct")
			require.Error(t, err)
			require.False(t, isAuthFailed(err), "got %v", err)

			// the IdP's details are for the server log only, never the device
			var withInternal platform_http.ErrWithInternal
			require.ErrorAs(t, err, &withInternal)
			require.Contains(t, withInternal.Internal(), "client secret")
			require.NotContains(t, err.Error(), "invalid_client")
			require.NotContains(t, err.Error(), "error_description")
		})
	}

	t.Run("config error reaches the device as a 400 without the IdP's details and is logged as an error", func(t *testing.T) {
		c := newClient(t, http.StatusUnauthorized,
			`{"error":"invalid_client","error_description":"The client secret supplied for a confidential client is invalid."}`)
		_, err := c.ValidatePasswordAndGetClaims(t.Context(), "user", "correct")
		require.Error(t, err)
		err = ctxerr.Wrap(t.Context(), err, "psso password validation")

		rec := httptest.NewRecorder()
		endpointer.EncodeError(t.Context(), err, rec, nil)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.NotContains(t, rec.Body.String(), "invalid_client")
		require.NotContains(t, rec.Body.String(), "client secret supplied")

		// client errors are logged at debug level, which would hide the
		// misconfiguration from the admin
		var buf bytes.Buffer
		logCtx := &logging.LoggingContext{}
		logCtx.SetErrs(err)
		logCtx.Log(t.Context(), slog.New(slog.NewJSONHandler(&buf, nil)))
		require.Contains(t, buf.String(), `"level":"ERROR"`)
		require.Contains(t, buf.String(), "check the Apple account provisioning token URL")
	})

	t.Run("success returns the id_token claims", func(t *testing.T) {
		idToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "abc", "email": "u@example.com"}).
			SignedString([]byte("test-key"))
		require.NoError(t, err)
		c := newClient(t, http.StatusOK, `{"id_token":"`+idToken+`","expires_in":3600}`)
		claims, err := c.ValidatePasswordAndGetClaims(t.Context(), "user", "correct")
		require.NoError(t, err)
		require.Equal(t, "abc", claims.Subject)
		require.Equal(t, 3600, claims.ExpiresIn)
	})
}

// TestPSSOOIDCROPGClientVerifyConfig replays IdP token responses from
// testdata/psso_idp/<provider>.json. To cover a new provider, add a file with its
// responses and the outcome each should produce.
func TestPSSOOIDCROPGClientVerifyConfig(t *testing.T) {
	const (
		urlField     = "mdm.apple_account_provisioning.oauth_idp_token_url"
		settingField = "mdm.apple_account_provisioning"
	)
	wants := map[string]struct{ field, message string }{
		"client_credentials": {settingField, "rejected the client ID or client secret"},
		"app_setup":          {settingField, "didn't let this app sign users in with a password"},
		"unexpected":         {settingField, "rejected Fleet's test sign-in"},
		"token_url":          {urlField, "didn't return an OAuth response (HTTP"},
		"token_url_token":    {urlField, "returned a token for " + ropgVerifyUsername},
		"token_url_status":   {urlField, "didn't return an OAuth error"},
		"token_url_endpoint": {urlField, "isn't your IdP's OAuth token endpoint"},
	}

	files, err := filepath.Glob(filepath.Join("testdata", "psso_idp", "*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		var fixture struct {
			Provider string `json:"provider"`
			Cases    []struct {
				Name        string `json:"name"`
				Status      int    `json:"status"`
				ContentType string `json:"content_type"`
				Body        string `json:"body"`
				Want        string `json:"want"`
			} `json:"cases"`
		}
		b, err := os.ReadFile(file)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &fixture))

		for _, tc := range fixture.Cases {
			t.Run(fixture.Provider+"/"+tc.Name, func(t *testing.T) {
				var form url.Values
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_ = r.ParseForm() //nolint:gosec // G120: test IdP
					form = r.PostForm
					contentType := tc.ContentType
					if contentType == "" {
						contentType = "application/json"
					}
					w.Header().Set("Content-Type", contentType)
					w.WriteHeader(tc.Status)
					_, _ = w.Write([]byte(tc.Body))
				}))
				t.Cleanup(srv.Close)

				err := PSSOOIDCROPGClient{TokenURL: srv.URL, ClientID: "client", ClientSecret: "secret", HTTPClient: srv.Client()}.VerifyConfig(t.Context())

				require.Equal(t, "password", form.Get("grant_type"))
				require.Equal(t, ropgVerifyUsername, form.Get("username"))
				require.Len(t, form.Get("password"), 32)
				require.Equal(t, "client", form.Get("client_id"))
				require.Equal(t, "secret", form.Get("client_secret"))

				if tc.Want == "ok" {
					require.NoError(t, err)
					return
				}
				want, ok := wants[tc.Want]
				require.True(t, ok, "unknown want %q in %s", tc.Want, file)
				var invalid *fleet.InvalidArgumentError
				require.ErrorAs(t, err, &invalid)
				require.Contains(t, err.Error(), want.field+" ")
				require.Contains(t, err.Error(), want.message)
				require.NotContains(t, err.Error(), "oidc ropg", "internal error wrapping shown to the admin")
			})
		}
	}
}

func TestPSSOOIDCROPGClientVerifyConfigUnreachable(t *testing.T) {
	verify := func(t *testing.T, tokenURL string, client *http.Client) string {
		err := PSSOOIDCROPGClient{TokenURL: tokenURL, ClientID: "client", ClientSecret: "secret", HTTPClient: client}.VerifyConfig(t.Context())
		var invalid *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invalid)
		require.Contains(t, err.Error(), "mdm.apple_account_provisioning.oauth_idp_token_url ")
		// Go's raw network errors name resolvers and internal hosts.
		for _, raw := range []string{"dial tcp", `Post "`, "lookup ", "oidc ropg"} {
			require.NotContains(t, err.Error(), raw)
		}
		return err.Error()
	}

	t.Run("unknown host", func(t *testing.T) {
		client := &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return nil, &net.DNSError{Err: "no such host", Name: "idp.invalid", Server: "10.0.0.53:53", IsNotFound: true}
			},
		}}
		msg := verify(t, "https://idp.invalid/oauth2/v1/token", client)
		require.Contains(t, msg, "Fleet couldn't find the host idp.invalid.")
		require.NotContains(t, msg, "10.0.0.53")
	})

	t.Run("connection refused", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()
		msg := verify(t, srv.URL, nil)
		require.Contains(t, msg, "Fleet couldn't connect to "+strings.TrimPrefix(srv.URL, "http://"))
	})

	t.Run("untrusted certificate", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.NotFoundHandler())
		t.Cleanup(srv.Close)
		msg := verify(t, srv.URL, nil)
		require.Contains(t, msg, "Fleet couldn't verify the TLS certificate of "+strings.TrimPrefix(srv.URL, "https://"))
	})

	t.Run("IdP slower than the timeout", func(t *testing.T) {
		orig := ropgVerifyTimeout
		ropgVerifyTimeout = 50 * time.Millisecond
		t.Cleanup(func() { ropgVerifyTimeout = orig })

		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-release
		}))
		t.Cleanup(srv.Close)
		t.Cleanup(func() { close(release) })

		msg := verify(t, srv.URL, srv.Client())
		require.Contains(t, msg, "The token URL didn't respond within 50ms.")
	})
}
