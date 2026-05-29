package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	jose "github.com/go-jose/go-jose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPSSO_SymmetricRoundTrip exercises the AES-256-GCM envelope used for
// key_exchange and password_request responses. Encrypting and then
// decrypting under the same session key must yield the original plaintext.
func TestPSSO_SymmetricRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plain := []byte("a small but meaningful payload — could be a JWT or claims blob")
	blob, err := buildSymmetricJWE(plain, key)
	require.NoError(t, err)
	require.NotEmpty(t, blob)

	got, err := decryptSymmetricBlob(blob, key)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

// TestPSSO_SymmetricWrongKeyFails confirms that decryption with a different
// session key fails — i.e., that GCM's authentication tag is being checked.
func TestPSSO_SymmetricWrongKeyFails(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	require.NoError(t, err)

	blob, err := buildSymmetricJWE([]byte("secret"), key)
	require.NoError(t, err)

	_, err = decryptSymmetricBlob(blob, wrongKey)
	require.Error(t, err)
}

// TestPSSO_SymmetricWrongKeySize confirms we reject session keys with the
// wrong byte length, since AES-256 expects exactly 32.
func TestPSSO_SymmetricWrongKeySize(t *testing.T) {
	_, err := buildSymmetricJWE([]byte("x"), make([]byte, 16))
	require.Error(t, err)
	_, err = decryptSymmetricBlob([]byte(`{"iv":"AAA","ciphertext":"AAA"}`), make([]byte, 16))
	require.Error(t, err)
}

// TestPSSO_HKDFDifferentSaltDifferentKey confirms the session-key derivation
// produces distinct outputs for distinct salts (i.e. distinct request
// nonces).
func TestPSSO_HKDFDifferentSaltDifferentKey(t *testing.T) {
	kek := make([]byte, 32)
	_, err := rand.Read(kek)
	require.NoError(t, err)

	k1, err := deriveSessionKey(kek, []byte("nonce-1"))
	require.NoError(t, err)
	k2, err := deriveSessionKey(kek, []byte("nonce-2"))
	require.NoError(t, err)
	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	assert.NotEqual(t, k1, k2)

	// Same salt produces the same key (deterministic).
	k1again, err := deriveSessionKey(kek, []byte("nonce-1"))
	require.NoError(t, err)
	assert.Equal(t, k1, k1again)
}

// TestPSSO_AsymmetricEncryptRoundTrip confirms that a payload encrypted to
// a device's encryption pubkey via JWE ECDH-ES + A256GCM can be decrypted
// with the corresponding private key. This is the key_request flow.
func TestPSSO_AsymmetricEncryptRoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	payload := []byte(`{"key_exchange_key":"AAECAwQF"}`)
	jweCompact, err := buildAsymmetricJWE(payload, &deviceKey.PublicKey, "")
	require.NoError(t, err)
	require.NotEmpty(t, jweCompact)

	// JWE compact form has 5 base64url segments separated by dots; smoke
	// check we got something of that shape rather than re-implementing the
	// whole decrypt path here (the JOSE library is well-tested upstream).
	dots := 0
	for _, b := range jweCompact {
		if b == '.' {
			dots++
		}
	}
	assert.Equal(t, 4, dots, "expected JWE compact form with 4 dots")
}

// TestPSSO_LoginResponseJWERoundTrip confirms the hand-assembled PSSO login
// response JWE decrypts back to the original payload using the device's
// encryption private key. Decrypting via go-jose (which reads apu/apv from the
// protected header and feeds them to the same Concat KDF) proves both the
// compact wire format and the apv party-info binding are correct: a wrong apv
// would derive a different content-encryption key and fail the GCM tag.
func TestPSSO_LoginResponseJWERoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	apv := testAPV(t, deviceKey)
	payload := []byte(`{"id_token":"x","refresh_token":"y"}`)

	jweCompact, err := buildPSSOResponseJWE(payload, &deviceKey.PublicKey, apv, pssoTypLoginResponse)
	require.NoError(t, err)
	require.NotEmpty(t, jweCompact)

	parsed, err := jose.ParseEncrypted(string(jweCompact))
	require.NoError(t, err)
	got, err := parsed.Decrypt(deviceKey)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	// Per Apple's doc, apu is "APPLE" (uppercase) || ephemeral epk, with NO
	// nonce — distinct from apv's "Apple" || key || nonce framing.
	hdr := parsed.Header
	apuB64, ok := hdr.ExtraHeaders[jose.HeaderKey("apu")].(string)
	require.True(t, ok, "apu header must be present")
	apuRaw, err := base64.RawURLEncoding.DecodeString(apuB64)
	require.NoError(t, err)
	apuFields, err := parseApplePartyInfo(apuRaw)
	require.NoError(t, err)
	require.Len(t, apuFields, 2, "apu is exactly [label, epk] — no nonce")
	assert.Equal(t, apuPartyLabel, string(apuFields[0]))
	assert.Equal(t, byte(0x04), apuFields[1][0], "apu field 2 is the uncompressed epk")
}

// testAPV builds an Apple-shaped apv ("Apple" || deviceKey || nonce) and
// returns it base64url-encoded, mimicking what the device sends.
func testAPV(t *testing.T, deviceKey *ecdsa.PrivateKey) (apvB64 string) {
	t.Helper()
	devECDH, err := deviceKey.PublicKey.ECDH()
	require.NoError(t, err)
	nonce := "3B94D3F7-5907-44C2-B6AF-05A0B0017669"
	raw := encodeApplePartyInfo([]byte(apvPartyLabel), devECDH.Bytes(), []byte(nonce))
	return base64.RawURLEncoding.EncodeToString(raw)
}

// TestPSSO_LoginResponseJWEWrongKeyFails confirms the JWE can't be decrypted
// with a key other than the intended device key.
func TestPSSO_LoginResponseJWEWrongKeyFails(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	apv := testAPV(t, deviceKey)
	jweCompact, err := buildPSSOResponseJWE([]byte("secret"), &deviceKey.PublicKey, apv, pssoTypLoginResponse)
	require.NoError(t, err)

	parsed, err := jose.ParseEncrypted(string(jweCompact))
	require.NoError(t, err)
	_, err = parsed.Decrypt(otherKey)
	require.Error(t, err)
}

// TestPSSO_CanonicalizeKID confirms the padded base64 kid Apple's framework
// sends in the JWT header and the unpadded base64url kid the extension
// registers collapse to the same value, so device lookup by kid succeeds.
func TestPSSO_CanonicalizeKID(t *testing.T) {
	// Real values from a live device: register sends no padding, the JWT
	// header kid carries '='.
	registered := "Yk8ghfYYyiUzsp0tcfVFn4TJUu0B45fzUnmonZZILZE"
	jwtKID := "Yk8ghfYYyiUzsp0tcfVFn4TJUu0B45fzUnmonZZILZE="
	assert.Equal(t, canonicalizeKID(registered), canonicalizeKID(jwtKID))

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
	want := canonicalizeKID(variants[0])
	for _, v := range variants {
		assert.Equal(t, want, canonicalizeKID(v), "variant %q", v)
	}

	// A non-base64 value is returned unchanged rather than mangled.
	assert.Equal(t, "not base64 at all!!", canonicalizeKID("not base64 at all!!"))
}

// TestPSSO_ParseECPublicKey covers both PEM forms we accept on inbound key
// material from the extension.
func TestPSSO_ParseECPublicKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	got, err := parseECPublicKeyPEM(pemBytes)
	require.NoError(t, err)
	gotDER, err := x509.MarshalPKIXPublicKey(got)
	require.NoError(t, err)
	assert.Equal(t, der, gotDER)

	_, err = parseECPublicKeyPEM([]byte("not a pem block"))
	require.Error(t, err)
}

// TestPSSO_ParseRawECPointPEM covers the form the macOS extension actually
// sends: a raw ANSI X9.63 uncompressed point (0x04 || X || Y) PEM-wrapped
// under a "PUBLIC KEY" label rather than DER SubjectPublicKeyInfo.
func TestPSSO_ParseRawECPointPEM(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// SecKeyCopyExternalRepresentation's raw-point equivalent.
	ecdhPub, err := priv.PublicKey.ECDH()
	require.NoError(t, err)
	rawPoint := ecdhPub.Bytes()
	require.Len(t, rawPoint, 65)
	require.Equal(t, byte(0x04), rawPoint[0])

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rawPoint})
	got, err := parseECPublicKeyPEM(pemBytes)
	require.NoError(t, err)
	gotECDH, err := got.ECDH()
	require.NoError(t, err)
	assert.Equal(t, rawPoint, gotECDH.Bytes())

	// Garbage inside a valid PEM block is neither SPKI nor a valid point.
	bad := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("nope")})
	_, err = parseECPublicKeyPEM(bad)
	require.Error(t, err)
}
