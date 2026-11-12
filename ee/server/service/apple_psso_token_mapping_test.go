package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signTestIDToken builds an id_token JWT for parseOIDCIDTokenClaims. The
// signature isn't verified there, so any HS256 key works.
func signTestIDToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-key"))
	require.NoError(t, err)
	return signed
}

func TestParseOIDCIDTokenClaims(t *testing.T) {
	t.Run("captures custom claims in Extra and excludes reserved/typed claims", func(t *testing.T) {
		idToken := signTestIDToken(t, jwt.MapClaims{
			"sub":                "00u123",
			"email":              "fleetie@example.com",
			"name":               "Fleetie Example",
			"preferred_username": "fleetie@example.com",
			"accountUsername":    "fleetie",
			"AccountFullName":    "Fleetie E.",
			"department":         "engineering",
			// Registered claims Fleet overwrites when re-signing; must not leak into Extra.
			"iss":   "https://idp.example.com",
			"aud":   "okta-client",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"nonce": "idp-nonce",
		})

		claims, err := parseOIDCIDTokenClaims(idToken)
		require.NoError(t, err)

		assert.Equal(t, "00u123", claims.Subject)
		assert.Equal(t, "fleetie@example.com", claims.Email)
		assert.Equal(t, "Fleetie Example", claims.Name)
		assert.Equal(t, "fleetie@example.com", claims.PreferredUsername)

		// Extra holds all non-reserved, non-typed claims regardless of prefix;
		// the account-prefix filter is applied later at mint time.
		assert.Equal(t, "fleetie", claims.Extra["accountUsername"])
		assert.Equal(t, "Fleetie E.", claims.Extra["AccountFullName"])
		assert.Equal(t, "engineering", claims.Extra["department"])

		for _, k := range []string{"sub", "email", "name", "preferred_username", "iss", "aud", "exp", "iat", "nonce"} {
			_, ok := claims.Extra[k]
			assert.Falsef(t, ok, "reserved/typed claim %q should not appear in Extra", k)
		}
	})

	t.Run("missing sub is an error", func(t *testing.T) {
		idToken := signTestIDToken(t, jwt.MapClaims{"email": "fleetie@example.com"})
		_, err := parseOIDCIDTokenClaims(idToken)
		require.Error(t, err)
	})
}

func TestBuildPSSOIDTokenClaims(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	expiresIn := 3600

	idpClaims := &fleet.PSSOClaims{
		Subject:           "00u123",
		Email:             "fleetie@example.com",
		Name:              "Fleetie Example",
		PreferredUsername: "fleetie@example.com",
		Extra: map[string]any{
			"accountUsername": "fleetie",     // forwarded (account prefix)
			"AccountFullName": "Fleetie E.",  // forwarded (case-insensitive prefix)
			"department":      "engineering", // dropped (no account prefix)
			// An IdP that tries to smuggle reserved claims must not win.
			"iss": "https://evil.example.com",
			"exp": int64(1),
		},
	}

	got := buildPSSOIDTokenClaims(idpClaims, "fleet.example.com", "okta-client", "device-nonce", now, expiresIn)

	// Standard identity claims always present.
	assert.Equal(t, "fleetie@example.com", got["email"])
	assert.Equal(t, "Fleetie Example", got["name"])
	assert.Equal(t, "fleetie@example.com", got["preferred_username"])

	// Namespaced custom claims forwarded; non-namespaced dropped.
	assert.Equal(t, "fleetie", got["accountUsername"])
	assert.Equal(t, "Fleetie E.", got["AccountFullName"])
	_, hasDepartment := got["department"]
	assert.False(t, hasDepartment, "non-account-prefixed claim should not be forwarded")

	// Fleet-controlled claims are authoritative; the IdP's attempts are overridden.
	assert.Equal(t, "fleet.example.com", got["iss"])
	assert.Equal(t, "00u123", got["sub"])
	assert.Equal(t, "okta-client", got["aud"])
	assert.Equal(t, "device-nonce", got["nonce"])
	assert.Equal(t, now.Unix(), got["iat"])
	assert.Equal(t, now.Add(time.Duration(expiresIn)*time.Second).Unix(), got["exp"])
}
