package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPSSONonceEndpointAcceptsFormBody guards against the nonce request struct
// losing its DecodeBody: Apple's AppSSOAgent POSTs a urlencoded
// grant_type=srv_challenge form, and without a body-decoder the framework
// falls through to JSON decoding and rejects it with a 400.
func TestPSSONonceEndpointAcceptsFormBody(t *testing.T) {
	decode := makeDecoder(pssoNonceRequest{}, 1<<20)

	for _, body := range []string{"grant_type=srv_challenge", ""} {
		r := httptest.NewRequest("POST", "/api/mdm/apple/psso/nonce", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err := decode(t.Context(), r)
		require.NoError(t, err, "body %q", body)
	}
}

func TestPSSORegistrationRequestDecodeBody(t *testing.T) {
	pubPEM := "-----BEGIN PUBLIC KEY-----\nMFkw+abc/def=\n-----END PUBLIC KEY-----"
	form := url.Values{}
	form.Set("device_uuid", "A72B07D0-2E08-45CE-9423-1FCAFFAEC390")
	form.Set("device_signing_key", pubPEM)
	form.Set("device_encryption_key", pubPEM)
	form.Set("signing_key_id", "sign-kid")
	form.Set("encryption_key_id", "enc-kid")

	var req pssoRegistrationRequest
	err := req.DecodeBody(t.Context(), strings.NewReader(form.Encode()), nil, nil)
	require.NoError(t, err)
	require.Equal(t, "A72B07D0-2E08-45CE-9423-1FCAFFAEC390", req.DeviceUUID)
	// PEM survives urlencoding round trip: '+', '/', '=' and newlines intact.
	require.Equal(t, pubPEM, req.DeviceSigningKey)
	require.Equal(t, pubPEM, req.DeviceEncryptionKey)
	require.Equal(t, "sign-kid", req.SigningKeyID)
	require.Equal(t, "enc-kid", req.EncryptionKeyID)
}

func TestPSSOTokenRequestDecodeBody(t *testing.T) {
	t.Run("extracts assertion", func(t *testing.T) {
		form := url.Values{}
		form.Set("assertion", "eyJhbGciOiJFUzI1NiJ9.payload.sig")

		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader(form.Encode()), nil, nil)
		require.NoError(t, err)
		require.Equal(t, "eyJhbGciOiJFUzI1NiJ9.payload.sig", req.Assertion)
	})

	t.Run("missing assertion rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader("other=value"), nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing assertion")
	})

	t.Run("empty body rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader(""), nil, nil)
		require.Error(t, err)
	})

	t.Run("malformed form rejected", func(t *testing.T) {
		var req pssoTokenRequest
		err := req.DecodeBody(t.Context(), strings.NewReader("a=%zz"), nil, nil)
		require.Error(t, err)
	})
}

type pssoTestNotFoundError struct{}

func (pssoTestNotFoundError) Error() string    { return "not found" }
func (pssoTestNotFoundError) IsNotFound() bool { return true }

// pssoBootstrapMock wires a mock datastore over an in-memory asset map so the
// bootstrap can be exercised without MySQL. GetAll returns a not-found error
// when nothing matches (mirroring the real datastore) and Insert appends.
func pssoBootstrapMock(store map[fleet.MDMAssetName]fleet.MDMConfigAsset) *mock.DataStore {
	ds := new(mock.DataStore)
	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		out := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		for _, n := range names {
			if a, ok := store[n]; ok {
				out[n] = a
			}
		}
		if len(out) == 0 {
			return nil, pssoTestNotFoundError{}
		}
		if len(out) < len(names) {
			return out, errors.New("partial result")
		}
		return out, nil
	}
	ds.InsertMDMConfigAssetsFunc = func(_ context.Context, assets []fleet.MDMConfigAsset, _ sqlx.ExtContext) error {
		for _, a := range assets {
			store[a.Name] = a
		}
		return nil
	}
	return ds
}

func parsePEMSigningKey(t *testing.T, value []byte) *ecdsa.PrivateKey {
	t.Helper()
	block, _ := pem.Decode(value)
	require.NotNil(t, block)
	key, err := x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)
	return key
}

func parsePEMCert(t *testing.T, value []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(value)
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

func TestBootstrapPSSOAssets(t *testing.T) {
	ctx := context.Background()

	t.Run("creates signing key, CA, and encryption key when all absent", func(t *testing.T) {
		store := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		ds := pssoBootstrapMock(store)

		require.NoError(t, bootstrapPSSOAssets(ctx, ds))
		require.True(t, ds.InsertMDMConfigAssetsFuncInvoked)
		require.Contains(t, store, fleet.MDMAssetPSSOSigningKey)
		require.Contains(t, store, fleet.MDMAssetPSSOCACert)
		require.Contains(t, store, fleet.MDMAssetPSSOEncryptionKey)

		signingKey := parsePEMSigningKey(t, store[fleet.MDMAssetPSSOSigningKey].Value)
		caCert := parsePEMCert(t, store[fleet.MDMAssetPSSOCACert].Value)
		encKey := parsePEMSigningKey(t, store[fleet.MDMAssetPSSOEncryptionKey].Value)

		assert.True(t, caCert.IsCA)
		// The CA is self-signed by the signing key, so its public key is the
		// signing key's public key.
		caPub, ok := caCert.PublicKey.(*ecdsa.PublicKey)
		require.True(t, ok)
		assert.True(t, caPub.Equal(&signingKey.PublicKey))
		require.NoError(t, caCert.CheckSignatureFrom(caCert))
		assert.WithinDuration(t, time.Now().AddDate(pssoCAValidYears, 0, 0), caCert.NotAfter, 24*time.Hour)

		// The encryption key is distinct from the signing key: NIST SP 800-57
		// forbids using one key for both signing and encryption.
		assert.False(t, encKey.PublicKey.Equal(&signingKey.PublicKey))
	})

	t.Run("no-op when all already exist", func(t *testing.T) {
		store := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		require.NoError(t, bootstrapPSSOAssets(ctx, pssoBootstrapMock(store)))
		seededKey := store[fleet.MDMAssetPSSOSigningKey].Value
		seededCA := store[fleet.MDMAssetPSSOCACert].Value
		seededEnc := store[fleet.MDMAssetPSSOEncryptionKey].Value

		ds := pssoBootstrapMock(store)
		require.NoError(t, bootstrapPSSOAssets(ctx, ds))
		// Nothing re-inserted, and the existing assets are untouched.
		assert.False(t, ds.InsertMDMConfigAssetsFuncInvoked)
		assert.Equal(t, seededKey, store[fleet.MDMAssetPSSOSigningKey].Value)
		assert.Equal(t, seededCA, store[fleet.MDMAssetPSSOCACert].Value)
		assert.Equal(t, seededEnc, store[fleet.MDMAssetPSSOEncryptionKey].Value)
	})

	t.Run("creates only the encryption key when signing and CA already exist", func(t *testing.T) {
		store := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		require.NoError(t, bootstrapPSSOAssets(ctx, pssoBootstrapMock(store)))
		existingKeyPEM := store[fleet.MDMAssetPSSOSigningKey].Value
		existingCAPEM := store[fleet.MDMAssetPSSOCACert].Value
		// Simulate a deployment configured before the encryption key existed.
		delete(store, fleet.MDMAssetPSSOEncryptionKey)

		ds := pssoBootstrapMock(store)
		require.NoError(t, bootstrapPSSOAssets(ctx, ds))
		require.True(t, ds.InsertMDMConfigAssetsFuncInvoked)

		// The signing key and CA are preserved; only the encryption key is minted.
		assert.Equal(t, existingKeyPEM, store[fleet.MDMAssetPSSOSigningKey].Value)
		assert.Equal(t, existingCAPEM, store[fleet.MDMAssetPSSOCACert].Value)
		require.Contains(t, store, fleet.MDMAssetPSSOEncryptionKey)
		encKey := parsePEMSigningKey(t, store[fleet.MDMAssetPSSOEncryptionKey].Value)
		signingKey := parsePEMSigningKey(t, existingKeyPEM)
		assert.False(t, encKey.PublicKey.Equal(&signingKey.PublicKey))
	})

	t.Run("creates only the CA over the existing key when CA is missing", func(t *testing.T) {
		store := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		// Seed a signing key only (e.g. a POC instance pre-dating the CA asset).
		require.NoError(t, bootstrapPSSOAssets(ctx, pssoBootstrapMock(store)))
		existingKeyPEM := store[fleet.MDMAssetPSSOSigningKey].Value
		delete(store, fleet.MDMAssetPSSOCACert)

		ds := pssoBootstrapMock(store)
		require.NoError(t, bootstrapPSSOAssets(ctx, ds))

		// The signing key is preserved (not regenerated) and the new CA is signed by it.
		assert.Equal(t, existingKeyPEM, store[fleet.MDMAssetPSSOSigningKey].Value)
		signingKey := parsePEMSigningKey(t, existingKeyPEM)
		caCert := parsePEMCert(t, store[fleet.MDMAssetPSSOCACert].Value)
		caPub, ok := caCert.PublicKey.(*ecdsa.PublicKey)
		require.True(t, ok)
		assert.True(t, caPub.Equal(&signingKey.PublicKey))
	})
}
