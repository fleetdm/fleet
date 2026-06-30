package pssocrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v3"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildAPV is a test helper mirroring what a PSSO client sends: an Apple-shaped
// apv ("Apple" || encKey || nonce), base64url-encoded.
func buildAPV(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	apv, err := BuildAPV(&key.PublicKey, []byte("3B94D3F7-5907-44C2-B6AF-05A0B0017669"))
	require.NoError(t, err)
	return apv
}

// TestAsymmetricEncryptRoundTrip confirms that a payload encrypted to a device's
// encryption pubkey via JWE ECDH-ES + A256GCM produces a valid compact JWE.
func TestAsymmetricEncryptRoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	payload := []byte(`{"claims":"AAECAwQF"}`)
	jweCompact, err := BuildAsymmetricJWE(payload, &deviceKey.PublicKey, "")
	require.NoError(t, err)
	require.NotEmpty(t, jweCompact)

	// JWE compact form has 5 base64url segments separated by dots; smoke check we
	// got something of that shape rather than re-implementing the whole decrypt
	// path here (the JOSE library is well-tested upstream).
	dots := 0
	for _, b := range jweCompact {
		if b == '.' {
			dots++
		}
	}
	assert.Equal(t, 4, dots, "expected JWE compact form with 4 dots")
}

// TestLoginResponseJWERoundTrip confirms the hand-assembled PSSO login response
// JWE decrypts back to the original payload using the device's encryption
// private key. Decrypting via go-jose (which reads apu/apv from the protected
// header and feeds them to the same Concat KDF) proves both the compact wire
// format and the apv party-info binding are correct: a wrong apv would derive a
// different content-encryption key and fail the GCM tag.
func TestLoginResponseJWERoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	apv := buildAPV(t, deviceKey)
	payload := []byte(`{"id_token":"x","refresh_token":"y"}`)

	jweCompact, err := BuildPartyInfoJWE(payload, &deviceKey.PublicKey, apv, TypLoginResponse)
	require.NoError(t, err)
	require.NotEmpty(t, jweCompact)

	parsed, err := jose.ParseEncrypted(string(jweCompact))
	require.NoError(t, err)
	got, err := parsed.Decrypt(deviceKey)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	// Per Apple's doc, apu is "APPLE" (uppercase) || ephemeral epk, with NO nonce
	// — distinct from apv's "Apple" || key || nonce framing.
	hdr := parsed.Header
	apuB64, ok := hdr.ExtraHeaders[jose.HeaderKey("apu")].(string)
	require.True(t, ok, "apu header must be present")
	apuRaw, err := base64.RawURLEncoding.DecodeString(apuB64)
	require.NoError(t, err)
	apuFields, err := ParseApplePartyInfo(apuRaw)
	require.NoError(t, err)
	require.Len(t, apuFields, 2, "apu is exactly [label, epk] — no nonce")
	assert.Equal(t, APUPartyLabel, string(apuFields[0]))
	assert.Equal(t, byte(0x04), apuFields[1][0], "apu field 2 is the uncompressed epk")
}

// TestLoginResponseJWEWrongKeyFails confirms the JWE can't be decrypted with a
// key other than the intended device key.
func TestLoginResponseJWEWrongKeyFails(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	apv := buildAPV(t, deviceKey)
	jweCompact, err := BuildPartyInfoJWE([]byte("secret"), &deviceKey.PublicKey, apv, TypLoginResponse)
	require.NoError(t, err)

	parsed, err := jose.ParseEncrypted(string(jweCompact))
	require.NoError(t, err)
	_, err = parsed.Decrypt(otherKey)
	require.Error(t, err)
}

// TestKeyExchangeSharedSecretMatches confirms the unlock-key DH is symmetric:
// the server's ECDH(provisioned_priv, device_pub) equals the device's
// ECDH(device_priv, provisioned_pub).
func TestKeyExchangeSharedSecretMatches(t *testing.T) {
	provisioned, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	deviceDH, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Server side: what ComputeECDHShared does, against the device's public.
	deviceECDH, err := deviceDH.PublicKey.ECDH()
	require.NoError(t, err)
	serverShared, err := ComputeECDHShared(provisioned, deviceECDH.Bytes())
	require.NoError(t, err)
	require.Len(t, serverShared, 32)

	// Device side: ECDH(device_priv, provisioned_pub) — must match.
	provECDH, err := provisioned.PublicKey.ECDH()
	require.NoError(t, err)
	devPriv, err := deviceDH.ECDH()
	require.NoError(t, err)
	deviceShared, err := devPriv.ECDH(provECDH)
	require.NoError(t, err)
	assert.Equal(t, deviceShared, serverShared)
}

// TestTokenClaimsLeeway confirms inbound JWT time claims tolerate small clock
// skew between the Mac and the server: an iat slightly in the future (Mac clock
// ahead) or an exp slightly in the past must not fail validation, while skew
// beyond the leeway still does.
func TestTokenClaimsLeeway(t *testing.T) {
	now := time.Now()
	claimsAt := func(iat, exp time.Time) *TokenClaims {
		return &TokenClaims{RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
		}}
	}

	// In sync: valid.
	require.NoError(t, claimsAt(now, now.Add(5*time.Minute)).Valid())

	// Mac clock slightly ahead: iat in the (server's) future, within leeway.
	require.NoError(t, claimsAt(now.Add(30*time.Second), now.Add(5*time.Minute)).Valid())

	// exp just passed, within leeway.
	require.NoError(t, claimsAt(now.Add(-5*time.Minute), now.Add(-30*time.Second)).Valid())

	// Beyond leeway both ways.
	err := claimsAt(now.Add(JWTLeeway+time.Minute), now.Add(10*time.Minute)).Valid()
	require.ErrorIs(t, err, jwt.ErrTokenUsedBeforeIssued)
	err = claimsAt(now.Add(-10*time.Minute), now.Add(-JWTLeeway-time.Minute)).Valid()
	require.ErrorIs(t, err, jwt.ErrTokenExpired)

	// Absent time claims are not required (registration-era JWTs).
	require.NoError(t, (&TokenClaims{}).Valid())
}

// TestCanonicalizeKID confirms the padded base64 kid Apple's framework sends in
// the JWT header and the unpadded base64url kid the extension registers collapse
// to the same value, so device lookup by kid succeeds.
func TestCanonicalizeKID(t *testing.T) {
	// Real values from a live device: register sends no padding, the JWT header
	// kid carries '='.
	registered := "Yk8ghfYYyiUzsp0tcfVFn4TJUu0B45fzUnmonZZILZE"
	jwtKID := "Yk8ghfYYyiUzsp0tcfVFn4TJUu0B45fzUnmonZZILZE="
	assert.Equal(t, CanonicalizeKID(registered), CanonicalizeKID(jwtKID))

	// 32 random bytes encoded every which way must all canonicalize equal.
	raw := make([]byte, 32)
	_, err := rand.Read(raw)
	require.NoError(t, err)
	variants := []string{
		base64.RawURLEncoding.EncodeToString(raw),
		base64.URLEncoding.EncodeToString(raw),
		base64.RawStdEncoding.EncodeToString(raw),
		base64.StdEncoding.EncodeToString(raw),
	}
	want := CanonicalizeKID(variants[0])
	for _, v := range variants {
		assert.Equal(t, want, CanonicalizeKID(v), "variant %q", v)
	}

	// A non-base64 value is returned unchanged rather than mangled.
	assert.Equal(t, "not base64 at all!!", CanonicalizeKID("not base64 at all!!"))
}

// TestParseECPublicKey covers both PEM forms we accept on inbound key material
// from the extension.
func TestParseECPublicKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	got, err := ParseECPublicKeyPEM(pemBytes)
	require.NoError(t, err)
	gotDER, err := x509.MarshalPKIXPublicKey(got)
	require.NoError(t, err)
	assert.Equal(t, der, gotDER)

	_, err = ParseECPublicKeyPEM([]byte("not a pem block"))
	require.Error(t, err)
}

// TestParseRawECPointPEM covers the form the macOS extension actually sends: a
// raw ANSI X9.63 uncompressed point (0x04 || X || Y) PEM-wrapped under a "PUBLIC
// KEY" label rather than DER SubjectPublicKeyInfo.
func TestParseRawECPointPEM(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// SecKeyCopyExternalRepresentation's raw-point equivalent.
	rawPoint, err := RawECPoint(&priv.PublicKey)
	require.NoError(t, err)
	require.Len(t, rawPoint, 65)
	require.Equal(t, byte(0x04), rawPoint[0])

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rawPoint})
	got, err := ParseECPublicKeyPEM(pemBytes)
	require.NoError(t, err)
	gotRaw, err := RawECPoint(got)
	require.NoError(t, err)
	assert.Equal(t, rawPoint, gotRaw)

	// Garbage inside a valid PEM block is neither SPKI nor a valid point.
	bad := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("nope")})
	_, err = ParseECPublicKeyPEM(bad)
	require.Error(t, err)
}

// TestInboundAssertionDecryptRoundTrip confirms a party-info JWE built by one
// side decrypts on the other, and that the typ header is pinned so a response of
// one media type can't be replayed as another.
func TestInboundAssertionDecryptRoundTrip(t *testing.T) {
	encKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	apv := buildAPV(t, encKey)
	plaintext := []byte(`{"password":"hunter2","username":"foo"}`)

	jwe, err := BuildPartyInfoJWE(plaintext, &encKey.PublicKey, apv, TypEncryptedLoginAssertion)
	require.NoError(t, err)

	got, err := DecryptPartyInfoJWE(jwe, encKey, TypEncryptedLoginAssertion)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	// A different recipient key can't open it.
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, err = DecryptPartyInfoJWE(jwe, other, TypEncryptedLoginAssertion)
	require.Error(t, err)

	// The typ is pinned: a JWE of another media type is rejected even with the
	// right key, so a key/login response can't be replayed as a login assertion.
	wrongTyp, err := BuildPartyInfoJWE(plaintext, &encKey.PublicKey, apv, TypLoginResponse)
	require.NoError(t, err)
	_, err = DecryptPartyInfoJWE(wrongTyp, encKey, TypEncryptedLoginAssertion)
	require.ErrorContains(t, err, "unexpected typ")
}

// TestEmbeddedAssertionPasswordRoundTrip confirms the password survives a build
// → parse round trip, and that the parser also accepts a compact JWT and rejects
// non-JSON/JWT input. The username is read from the signed outer JWT elsewhere.
func TestEmbeddedAssertionPasswordRoundTrip(t *testing.T) {
	plaintext, err := BuildEmbeddedAssertionPlaintext("hunter2")
	require.NoError(t, err)
	pw, err := ParseEmbeddedAssertionPassword(plaintext)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", pw)

	t.Run("compact JWT", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
		body := base64.RawURLEncoding.EncodeToString([]byte(`{"password":"pw","username":"alice"}`))
		password, err := ParseEmbeddedAssertionPassword([]byte(header + "." + body + "."))
		require.NoError(t, err)
		assert.Equal(t, "pw", password)
	})

	t.Run("bare JSON object", func(t *testing.T) {
		password, err := ParseEmbeddedAssertionPassword([]byte(`{"password":"pw2","username":"bob"}`))
		require.NoError(t, err)
		assert.Equal(t, "pw2", password)
	})

	t.Run("neither JSON nor JWT is rejected", func(t *testing.T) {
		_, err := ParseEmbeddedAssertionPassword([]byte("not-json-not-jwt"))
		require.Error(t, err)
	})
}
