package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
		t.Run(tc.name+" is a server error, not an auth failure", func(t *testing.T) {
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

	t.Run("config error reaches the device as a 500 without the IdP's details", func(t *testing.T) {
		c := newClient(t, http.StatusUnauthorized,
			`{"error":"invalid_client","error_description":"The client secret supplied for a confidential client is invalid."}`)
		_, err := c.ValidatePasswordAndGetClaims(t.Context(), "user", "correct")
		require.Error(t, err)

		rec := httptest.NewRecorder()
		endpointer.EncodeError(t.Context(), ctxerr.Wrap(t.Context(), err, "psso password validation"), rec, nil)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.NotContains(t, rec.Body.String(), "invalid_client")
		require.NotContains(t, rec.Body.String(), "client secret supplied")
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
