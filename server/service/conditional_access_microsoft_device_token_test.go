package service

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEntraSigner signs Entra-like ID tokens with a key served by a local JWKS endpoint, which
// it sets as the endpoint used to verify Entra device tokens.
type testEntraSigner struct {
	key *rsa.PrivateKey
	kid string
}

func newTestEntraSigner(t *testing.T) *testEntraSigner {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	s := &testEntraSigner{key: key, kid: "test-kid-" + t.Name()}

	jwk, err := jwkset.NewJWKFromKey(key.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{KID: s.kid, ALG: jwkset.AlgRS256},
	})
	require.NoError(t, err)
	jwks := jwkset.NewMemoryStorage()
	require.NoError(t, jwks.KeyWrite(t.Context(), jwk))
	jwksJSON, err := jwks.JSONPublic(t.Context())
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON)
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride("FLEET_DEV_ENTRA_DEVICE_TOKEN_JWKS_URI", srv.URL, t)
	return s
}

func (s *testEntraSigner) sign(t *testing.T, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid
	signed, err := token.SignedString(s.key)
	require.NoError(t, err)
	return signed
}

func testEntraDeviceTokenClaims(tenantID, clientID, deviceID, upn, nonce string, now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"iss":                "https://login.microsoftonline.com/" + tenantID + "/v2.0",
		"aud":                clientID,
		"tid":                tenantID,
		"deviceid":           deviceID,
		"preferred_username": upn,
		"nonce":              nonce,
		"iat":                now.Unix(),
		"nbf":                now.Unix(),
		"exp":                now.Add(time.Hour).Unix(),
	}
}

func TestEntraDeviceNonce(t *testing.T) {
	n := entraDeviceNonce("secret", "5080D39E-D146-58C3-BD4F-1ACAA9546C59")
	assert.Equal(t, n, entraDeviceNonce("secret", "5080D39E-D146-58C3-BD4F-1ACAA9546C59"))
	assert.Regexp(t, `^5080D39E-D146-58C3-BD4F-1ACAA9546C59\.[A-Za-z0-9_-]+$`, n)
	assert.NotEqual(t, n, entraDeviceNonce("other-secret", "5080D39E-D146-58C3-BD4F-1ACAA9546C59"))
	assert.NotEqual(t, n, entraDeviceNonce("secret", "11111111-2222-3333-4444-555555555555"))
}

func TestVerifyEntraDeviceToken(t *testing.T) {
	const (
		tenantID = "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b"
		clientID = "d6f479f6-9b7c-459d-baa4-9146b15e93bb"
		deviceID = "c57f684e-4a62-4eb7-9429-a055431e44f4"
		upn      = "user@example.onmicrosoft.com"
		nonce    = "host-uuid.mac"
	)
	signer := newTestEntraSigner(t)
	now := time.Now()

	claims, err := verifyEntraDeviceToken(t.Context(), signer.sign(t, testEntraDeviceTokenClaims(tenantID, clientID, deviceID, upn, nonce, now)), tenantID, clientID, now)
	require.NoError(t, err)
	assert.Equal(t, deviceID, claims.DeviceID)
	assert.Equal(t, upn, claims.PreferredUsername)
	assert.Equal(t, nonce, claims.Nonce)

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		mutate  func(c jwt.MapClaims)
		token   func(c jwt.MapClaims) string
		errPart string
	}{
		"other tenant":        {mutate: func(c jwt.MapClaims) { c["tid"] = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d" }, errPart: "tenant mismatch"},
		"v1 issuer":           {mutate: func(c jwt.MapClaims) { c["iss"] = "https://sts.windows.net/" + tenantID + "/" }, errPart: "issuer mismatch"},
		"other audience":      {mutate: func(c jwt.MapClaims) { c["aud"] = "00000003-0000-0000-c000-000000000000" }, errPart: "audience mismatch"},
		"expired":             {mutate: func(c jwt.MapClaims) { c["exp"] = now.Add(-10 * time.Minute).Unix() }, errPart: "expired"},
		"issued too long ago": {mutate: func(c jwt.MapClaims) { c["iat"] = now.Add(-2 * time.Hour).Unix() }, errPart: "accepted window"},
		"issued in future":    {mutate: func(c jwt.MapClaims) { c["iat"] = now.Add(time.Hour).Unix() }, errPart: "accepted window"},
		"no device id":        {mutate: func(c jwt.MapClaims) { delete(c, "deviceid") }, errPart: "invalid device ID"},
		"bad device id":       {mutate: func(c jwt.MapClaims) { c["deviceid"] = "x' OR 1=1" }, errPart: "invalid device ID"},
		"empty":               {token: func(jwt.MapClaims) string { return "" }, errPart: "empty token"},
		"signed by other key": {
			token: func(c jwt.MapClaims) string {
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
				tok.Header["kid"] = signer.kid
				s, err := tok.SignedString(otherKey)
				require.NoError(t, err)
				return s
			},
			errPart: "verification error",
		},
		"unknown kid": {
			token: func(c jwt.MapClaims) string {
				tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
				tok.Header["kid"] = "unknown"
				s, err := tok.SignedString(signer.key)
				require.NoError(t, err)
				return s
			},
			errPart: "unknown signing key",
		},
		"HS256": {
			token: func(c jwt.MapClaims) string {
				tok := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
				tok.Header["kid"] = signer.kid
				s, err := tok.SignedString([]byte("secret"))
				require.NoError(t, err)
				return s
			},
			errPart: "signing method HS256 is invalid",
		},
		"none": {
			token: func(c jwt.MapClaims) string {
				tok := jwt.NewWithClaims(jwt.SigningMethodNone, c)
				tok.Header["kid"] = signer.kid
				s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
				require.NoError(t, err)
				return s
			},
			errPart: "signing method none is invalid",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := testEntraDeviceTokenClaims(tenantID, clientID, deviceID, upn, nonce, now)
			raw := ""
			if tc.token != nil {
				raw = tc.token(c)
			} else {
				tc.mutate(c)
				raw = signer.sign(t, c)
			}
			_, err := verifyEntraDeviceToken(t.Context(), raw, tenantID, clientID, now)
			require.ErrorContains(t, err, tc.errPart)
		})
	}
}
