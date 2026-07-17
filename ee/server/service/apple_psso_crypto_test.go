package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/pssocrypto"
	"github.com/fleetdm/fleet/v4/server/mock"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The symmetric wire-format crypto these flows build on (party-info JWE, kid
// canonicalization, EC parsing, the inbound claim types) is tested in
// server/mdm/apple/psso/pssocrypto. The tests here cover the server-only logic
// that wraps it: key_context sealing under Fleet's signing key, resolving a
// device key from the datastore, and the password/JWT login plumbing.

// TestPSSO_SymmetricRoundTrip exercises the AES-256-GCM envelope used to seal
// key_context blobs. Encrypting and then decrypting under the same session key
// must yield the original plaintext.
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

// TestPSSO_SymmetricWrongKeySize confirms we reject session keys with the wrong
// byte length, since AES-256 expects exactly 32.
func TestPSSO_SymmetricWrongKeySize(t *testing.T) {
	_, err := buildSymmetricJWE([]byte("x"), make([]byte, 16))
	require.Error(t, err)
	_, err = decryptSymmetricBlob([]byte(`{"iv":"AAA","ciphertext":"AAA"}`), make([]byte, 16))
	require.Error(t, err)
}

// TestPSSO_HKDFDifferentSaltDifferentKey confirms the session-key derivation
// produces distinct outputs for distinct salts (i.e. distinct request nonces).
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

// TestPSSO_KeyContextRoundTrip confirms a provisioned private key sealed into a
// key_context (key request) is recovered intact when opened (key exchange).
func TestPSSO_KeyContextRoundTrip(t *testing.T) {
	signing, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	kcKey, err := deriveKeyContextKey(signing)
	require.NoError(t, err)
	require.Len(t, kcKey, 32)

	provisioned, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	const hostUUID = "ABCD-1234-host-uuid"
	sealed, err := sealKeyContext(provisioned, hostUUID, pssoKeyPurposeUserUnlock, kcKey)
	require.NoError(t, err)

	kc, got, err := openKeyContext(sealed, kcKey)
	require.NoError(t, err)
	assert.Equal(t, hostUUID, kc.HostUUID)
	assert.Equal(t, pssoKeyPurposeUserUnlock, kc.KeyPurpose)
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
	_, _, err = openKeyContext(sealed, otherKC)
	require.Error(t, err)
}

// TestPSSO_InboundJWTAlgorithmPinned confirms the token endpoint accepts only
// ES256-signed device JWTs: an HS256 or "none" token presenting the same kid is
// rejected, closing the alg-confusion path.
func TestPSSO_InboundJWTAlgorithmPinned(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	spki, err := x509.MarshalPKIXPublicKey(&deviceKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: spki})

	const kid = "device-signing-kid"
	ds := new(mock.DataStore)
	svc := &Service{ds: ds, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ds.GetPSSOKeyFunc = func(_ context.Context, _ string) (*fleet.PSSOKey, error) {
		return &fleet.PSSOKey{KID: kid, HostUUID: "host", KeyType: fleet.PSSOKeyTypeSigning, PEM: string(pubPEM)}, nil
	}

	signed := func(method jwt.SigningMethod, key any) string {
		tok := jwt.NewWithClaims(method, &pssocrypto.TokenClaims{RequestType: pssocrypto.RequestKey})
		tok.Header["kid"] = kid
		s, err := tok.SignedString(key)
		require.NoError(t, err)
		return s
	}

	// A valid ES256 token from the registered device verifies.
	claims, gotKey, err := svc.parsePSSOInboundJWT(t.Context(), []byte(signed(jwt.SigningMethodES256, deviceKey)))
	require.NoError(t, err)
	assert.Equal(t, pssocrypto.RequestKey, claims.RequestType)
	assert.Equal(t, fleet.PSSOKeyTypeSigning, gotKey.KeyType)

	// An HS256 token sharing the same kid is rejected (alg confusion).
	_, _, err = svc.parsePSSOInboundJWT(t.Context(), []byte(signed(jwt.SigningMethodHS256, []byte("attacker-secret"))))
	require.Error(t, err)

	// An unsigned ("none") token is rejected.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, &pssocrypto.TokenClaims{RequestType: pssocrypto.RequestKey})
	none.Header["kid"] = kid
	noneStr, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, _, err = svc.parsePSSOInboundJWT(t.Context(), []byte(noneStr))
	require.Error(t, err)
}

// TestPSSO_ResolveEncryptionKey covers resolving the response-encryption key
// from a request's apv blob: the kid is recomputed as SHA-256 of the raw key
// bytes the device placed in apv (matching how the extension registers its
// kids), looked up, and validated as an encryption key belonging to the
// requesting host. When the kid lookup misses, the host's registered encryption
// keys are compared point-by-point as a fallback.
func TestPSSO_ResolveEncryptionKey(t *testing.T) {
	const hostUUID = "ABCDEFGH-0000-0000-0000-111111111111"

	encPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rawPoint, err := pssocrypto.RawECPoint(&encPriv.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rawPoint})

	kid, err := pssocrypto.KIDFromRawECPoint(&encPriv.PublicKey)
	require.NoError(t, err)

	apv, err := pssocrypto.BuildAPV(&encPriv.PublicKey, []byte("nonce"))
	require.NoError(t, err)

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

// TestPSSO_ResolveLoginPassword confirms the password is taken from the
// plaintext claim when present, and decrypted out of the embedded assertion
// (using Fleet's stored encryption key) when password encryption is enabled.
func TestPSSO_ResolveLoginPassword(t *testing.T) {
	encKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	encDER, err := x509.MarshalECPrivateKey(encKey)
	require.NoError(t, err)
	encPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encDER})

	newSvc := func() *Service {
		ds := new(mock.DataStore)
		ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			out := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
			for _, n := range names {
				if n == fleet.MDMAssetPSSOEncryptionKey {
					out[n] = fleet.MDMConfigAsset{Name: n, Value: encPEM}
				}
			}
			return out, nil
		}
		return &Service{ds: ds, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	}

	t.Run("plaintext password claim", func(t *testing.T) {
		pw, err := newSvc().resolvePSSOLoginPassword(t.Context(), &pssocrypto.TokenClaims{Password: "plain"})
		require.NoError(t, err)
		assert.Equal(t, "plain", pw)
	})

	t.Run("encrypted embedded assertion", func(t *testing.T) {
		apv, err := pssocrypto.BuildAPV(&encKey.PublicKey, []byte("nonce"))
		require.NoError(t, err)
		inner := []byte(`{"password":"secret","username":"carol"}`)
		jwe, err := pssocrypto.BuildPartyInfoJWE(inner, &encKey.PublicKey, apv, pssocrypto.TypEncryptedLoginAssertion)
		require.NoError(t, err)

		pw, err := newSvc().resolvePSSOLoginPassword(t.Context(), &pssocrypto.TokenClaims{
			GrantType: pssocrypto.GrantTypeJWTBearer,
			Assertion: string(jwe),
		})
		require.NoError(t, err)
		assert.Equal(t, "secret", pw)
	})
}
