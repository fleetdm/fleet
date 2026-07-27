package regtoken

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func TestMintAndValidateRoundTrip(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	const hostUUID = "A72B07D0-2E08-45CE-9423-1FCAFFAEC390"

	token, err := Mint(key, hostUUID, now)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	sub, err := Validate(token, &key.PublicKey, now)
	require.NoError(t, err)
	require.Equal(t, hostUUID, sub)

	// Still valid years later, before expiry.
	sub, err = Validate(token, &key.PublicKey, now.Add(4*365*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, hostUUID, sub)
}

func TestMintValidation(t *testing.T) {
	key := testKey(t)
	now := time.Now()

	_, err := Mint(nil, "uuid", now)
	require.Error(t, err)

	_, err = Mint(key, "", now)
	require.Error(t, err)
}

func TestValidateRejectsExpired(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	token, err := Mint(key, "uuid", now)
	require.NoError(t, err)

	_, err = Validate(token, &key.PublicKey, now.Add(DefaultValidity+time.Hour))
	require.Error(t, err)
}

func TestValidateRejectsFutureIssuedAt(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	token, err := Mint(key, "uuid", now)
	require.NoError(t, err)

	// Validating as if "now" were before the token was issued must fail.
	_, err = Validate(token, &key.PublicKey, now.Add(-time.Hour))
	require.Error(t, err)
}

func TestValidateRejectsMissingIssuedAt(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	claims := jwt.RegisteredClaims{
		Subject:   "uuid",
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		// No IssuedAt.
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := tok.SignedString(key)
	require.NoError(t, err)

	_, err = Validate(signed, &key.PublicKey, now)
	require.Error(t, err)
}

func TestValidateRejectsWrongKey(t *testing.T) {
	key := testKey(t)
	other := testKey(t)
	now := time.Now()

	token, err := Mint(key, "uuid", now)
	require.NoError(t, err)

	_, err = Validate(token, &other.PublicKey, now)
	require.Error(t, err)
}

func TestValidateRejectsWrongAudience(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	claims := jwt.RegisteredClaims{
		Subject:   "uuid",
		Audience:  jwt.ClaimStrings{"some-other-audience"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := tok.SignedString(key)
	require.NoError(t, err)

	_, err = Validate(signed, &key.PublicKey, now)
	require.Error(t, err)
}

func TestValidateRejectsWrongSigningMethod(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	// HS256 token must be rejected: WithValidMethods locks to ES256, blocking
	// the classic alg-confusion downgrade.
	claims := jwt.RegisteredClaims{
		Subject:   "uuid",
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("symmetric-secret"))
	require.NoError(t, err)

	_, err = Validate(signed, &key.PublicKey, now)
	require.Error(t, err)
}

func TestMintFromPEMMatchesMint(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})

	token, err := MintFromPEM(keyPEM, "uuid", now)
	require.NoError(t, err)

	sub, err := Validate(token, &key.PublicKey, now)
	require.NoError(t, err)
	require.Equal(t, "uuid", sub)
}
