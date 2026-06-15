package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	jose "github.com/go-jose/go-jose/v3"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPSSO_SymmetricRoundTrip exercises the AES-256-GCM envelope used to
// seal key_context blobs. Encrypting and then decrypting under the same
// session key must yield the original plaintext.
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
// a device's encryption pubkey via JWE ECDH-ES + A256GCM produces a valid
// compact JWE.
func TestPSSO_AsymmetricEncryptRoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	payload := []byte(`{"claims":"AAECAwQF"}`)
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

// TestPSSO_KeyContextRoundTrip confirms a provisioned private key sealed into
// a key_context (key request) is recovered intact when opened (key exchange).
func TestPSSO_KeyContextRoundTrip(t *testing.T) {
	signing, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	kcKey, err := deriveKeyContextKey(signing)
	require.NoError(t, err)
	require.Len(t, kcKey, 32)

	provisioned, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	sealed, err := sealKeyContext(provisioned, kcKey)
	require.NoError(t, err)

	got, err := openKeyContext(sealed, kcKey)
	require.NoError(t, err)
	want, err := x509.MarshalECPrivateKey(provisioned)
	require.NoError(t, err)
	gotDER, err := x509.MarshalECPrivateKey(got)
	require.NoError(t, err)
	assert.Equal(t, want, gotDER)

	// A different server key can't open it.
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	otherKC, err := deriveKeyContextKey(other)
	require.NoError(t, err)
	_, err = openKeyContext(sealed, otherKC)
	require.Error(t, err)
}

// TestPSSO_KeyExchangeSharedSecretMatches confirms the unlock-key DH is
// symmetric: the server's ECDH(provisioned_priv, device_pub) equals the
// device's ECDH(device_priv, provisioned_pub).
func TestPSSO_KeyExchangeSharedSecretMatches(t *testing.T) {
	provisioned, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	deviceDH, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Server side: what computeECDHShared does, against the device's public.
	deviceECDH, err := deviceDH.PublicKey.ECDH()
	require.NoError(t, err)
	serverShared, err := computeECDHShared(provisioned, deviceECDH.Bytes())
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

// TestPSSO_TokenClaimsLeeway confirms inbound JWT time claims tolerate small
// clock skew between the Mac and the server: an iat slightly in the future
// (Mac clock ahead) or an exp slightly in the past must not fail validation,
// while skew beyond the leeway still does.
func TestPSSO_TokenClaimsLeeway(t *testing.T) {
	now := time.Now()
	claimsAt := func(iat, exp time.Time) *pssoTokenClaims {
		return &pssoTokenClaims{RegisteredClaims: jwt.RegisteredClaims{
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
	err := claimsAt(now.Add(pssoJWTLeeway+time.Minute), now.Add(10*time.Minute)).Valid()
	require.ErrorIs(t, err, jwt.ErrTokenUsedBeforeIssued)
	err = claimsAt(now.Add(-10*time.Minute), now.Add(-pssoJWTLeeway-time.Minute)).Valid()
	require.ErrorIs(t, err, jwt.ErrTokenExpired)

	// Absent time claims are not required (registration-era JWTs).
	require.NoError(t, (&pssoTokenClaims{}).Valid())
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

// TestPSSO_ResolveEncryptionKey covers resolving the response-encryption key
// from a request's apv blob: the kid is recomputed as SHA-256 of the raw key
// bytes the device placed in apv (matching how the extension registers its
// kids), looked up, and validated as an encryption key belonging to the
// requesting host. When the kid lookup misses, the host's registered
// encryption keys are compared point-by-point as a fallback.
func TestPSSO_ResolveEncryptionKey(t *testing.T) {
	const hostUUID = "ABCDEFGH-0000-0000-0000-111111111111"

	encPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	encECDH, err := encPriv.PublicKey.ECDH()
	require.NoError(t, err)
	rawPoint := encECDH.Bytes()
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rawPoint})

	sum := sha256.Sum256(rawPoint)
	kid := canonicalizeKID(base64.RawURLEncoding.EncodeToString(sum[:]))

	apv := base64.RawURLEncoding.EncodeToString(
		encodeApplePartyInfo([]byte(apvPartyLabel), rawPoint, []byte("nonce")))

	newSvc := func() (*Service, *mock.DataStore) {
		ds := new(mock.DataStore)
		svc := &Service{ds: ds, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
		return svc, ds
	}
	registeredKey := &fleet.PSSOKey{
		KID:      kid,
		HostUUID: hostUUID,
		KeyType:  fleet.PSSOKeyTypeEncryption,
		PEM:      string(pemBytes),
	}

	t.Run("resolves by kid computed from apv", func(t *testing.T) {
		svc, ds := newSvc()
		ds.GetPSSOKeyFunc = func(ctx context.Context, gotKID string) (*fleet.PSSOKey, error) {
			require.Equal(t, kid, gotKID)
			return registeredKey, nil
		}
		pub, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID, apv)
		require.NoError(t, err)
		assert.True(t, pub.Equal(&encPriv.PublicKey))
	})

	t.Run("rejects a key registered to a different host", func(t *testing.T) {
		svc, ds := newSvc()
		ds.GetPSSOKeyFunc = func(ctx context.Context, _ string) (*fleet.PSSOKey, error) {
			other := *registeredKey
			other.HostUUID = "some-other-host"
			return &other, nil
		}
		_, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID, apv)
		require.Error(t, err)
	})

	t.Run("rejects a signing key", func(t *testing.T) {
		svc, ds := newSvc()
		ds.GetPSSOKeyFunc = func(ctx context.Context, _ string) (*fleet.PSSOKey, error) {
			other := *registeredKey
			other.KeyType = fleet.PSSOKeyTypeSigning
			return &other, nil
		}
		_, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID, apv)
		require.Error(t, err)
	})

	t.Run("falls back to comparing the host's registered keys", func(t *testing.T) {
		svc, ds := newSvc()
		ds.GetPSSOKeyFunc = func(ctx context.Context, _ string) (*fleet.PSSOKey, error) {
			return nil, &testNotFoundError{}
		}
		ds.ListPSSOKeysFunc = func(ctx context.Context, gotUUID string) ([]*fleet.PSSOKey, error) {
			require.Equal(t, hostUUID, gotUUID)
			return []*fleet.PSSOKey{registeredKey}, nil
		}
		pub, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID, apv)
		require.NoError(t, err)
		assert.True(t, pub.Equal(&encPriv.PublicKey))
		assert.True(t, ds.ListPSSOKeysFuncInvoked)
	})

	t.Run("rejects when no registered key matches", func(t *testing.T) {
		svc, ds := newSvc()
		ds.GetPSSOKeyFunc = func(ctx context.Context, _ string) (*fleet.PSSOKey, error) {
			return nil, &testNotFoundError{}
		}
		ds.ListPSSOKeysFunc = func(ctx context.Context, _ string) ([]*fleet.PSSOKey, error) {
			return nil, nil
		}
		_, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID, apv)
		require.Error(t, err)
	})

	t.Run("rejects a malformed apv", func(t *testing.T) {
		svc, _ := newSvc()
		_, err := svc.resolvePSSOEncryptionKey(t.Context(), hostUUID,
			base64.RawURLEncoding.EncodeToString([]byte("not party info")))
		require.Error(t, err)
	})
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
