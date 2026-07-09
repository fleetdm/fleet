package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/regtoken"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pssoTestConfig describes the resolved configuration the PSSO flows read:
// public IdP fields + Fleet server URL from AppConfig, and the IdP client
// secret from mdm_config_assets. The feature is "configured" only when all of
// them are present.
type pssoTestConfig struct {
	serverURL string
	tokenURL  string
	clientID  string
	secret    string // empty => no stored secret asset
}

func configuredPSSOTestConfig() pssoTestConfig {
	return pssoTestConfig{ //nolint:gosec // G101: test value only, not a real credential
		serverURL: "https://fleet.example.com",
		tokenURL:  "https://idp.example.com/oauth2/v1/token",
		clientID:  "client-id",
		secret:    "client-secret",
	}
}

func newPSSOTestService(t *testing.T, cfg pssoTestConfig) (*Service, context.Context) {
	t.Helper()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.ServerSettings.ServerURL = cfg.serverURL
		ac.MDM.AppleAccountProvisioning = fleet.AppleAccountProvisioning{
			OAuthIdPTokenURL: optjson.SetString(cfg.tokenURL),
			OAuthIdPClientID: optjson.SetString(cfg.clientID),
		}
		return ac, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		out := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		if cfg.secret != "" {
			out[fleet.MDMAssetAppleAccountProvisioningIdPClientSecret] = fleet.MDMConfigAsset{
				Name:  fleet.MDMAssetAppleAccountProvisioningIdPClientSecret,
				Value: []byte(cfg.secret),
			}
		}
		return out, nil
	}
	auth, err := authz.NewAuthorizer()
	require.NoError(t, err)
	ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
	return &Service{
		ds:     ds,
		authz:  auth,
		logger: slog.New(slog.DiscardHandler),
	}, ctx
}

// memNonceStore is a minimal in-memory fleet.PSSONonceStore.
type memNonceStore struct {
	nonces map[string]struct{}
}

func (s *memNonceStore) Store(_ context.Context, nonce string, _ time.Duration) error {
	if s.nonces == nil {
		s.nonces = map[string]struct{}{}
	}
	s.nonces[nonce] = struct{}{}
	return nil
}

func (s *memNonceStore) Consume(_ context.Context, nonce string) (bool, error) {
	if _, ok := s.nonces[nonce]; !ok {
		return false, nil
	}
	delete(s.nonces, nonce)
	return true, nil
}

func TestPSSO_EndpointsGatedOnConfiguration(t *testing.T) {
	configured := configuredPSSOTestConfig()
	// When the public config is incomplete the feature is off for every
	// endpoint, determined without reading the client secret.
	notConfigured := []pssoTestConfig{
		{},
		func() pssoTestConfig { c := configured; c.serverURL = ""; return c }(),
		func() pssoTestConfig { c := configured; c.tokenURL = ""; return c }(),
		func() pssoTestConfig { c := configured; c.clientID = ""; return c }(),
	}

	for _, cfg := range notConfigured {
		svc, ctx := newPSSOTestService(t, cfg)

		// Device-facing endpoints return a 400.
		_, err := svc.PSSONonce(ctx)
		require.ErrorIs(t, err, errPSSONotConfigured)
		err = svc.PSSORegisterDevice(ctx, fleet.PSSODeviceRegistrationRequest{})
		require.ErrorIs(t, err, errPSSONotConfigured)
		_, err = svc.PSSOToken(ctx, []byte("ignored"))
		require.ErrorIs(t, err, errPSSONotConfigured)

		// Discovery endpoints return a 404.
		_, err = svc.PSSOJWKS(ctx)
		require.True(t, fleet.IsNotFound(err), "jwks should be 404, got %v", err)
		_, err = svc.PSSOAASA(ctx)
		require.True(t, fleet.IsNotFound(err), "aasa should be 404, got %v", err)
	}

	// The client secret is only required by the token (password login) flow, so
	// a missing secret gates that endpoint alone — the others don't read it.
	t.Run("token gated when secret missing", func(t *testing.T) {
		cfg := configured
		cfg.secret = ""
		svc, ctx := newPSSOTestService(t, cfg)
		_, err := svc.PSSOToken(ctx, []byte("ignored"))
		require.ErrorIs(t, err, errPSSONotConfigured)
	})
}

func TestPSSO_NonceIssuedAndConsumedWhenConfigured(t *testing.T) {
	svc, ctx := newPSSOTestService(t, configuredPSSOTestConfig())
	svc.pssoNonceStore = &memNonceStore{}

	nonce, err := svc.PSSONonce(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, nonce)

	// The nonce flow doesn't need the IdP client secret, so it must not pay the
	// mdm_config_assets read.
	require.False(t, svc.ds.(*mock.Store).GetAllMDMConfigAssetsByNameFuncInvoked)

	// First consume succeeds, replay is rejected.
	require.NoError(t, svc.consumePSSORequestNonce(ctx, nonce))
	err = svc.consumePSSORequestNonce(ctx, nonce)
	require.Error(t, err)
	var bre *fleet.BadRequestError
	require.ErrorAs(t, err, &bre)

	// A nonce Fleet never issued is rejected.
	err = svc.consumePSSORequestNonce(ctx, "never-issued")
	require.Error(t, err)
	// And the claim is required at all.
	err = svc.consumePSSORequestNonce(ctx, "")
	require.Error(t, err)
}

// mustDevicePSSOKey returns a fresh P-256 public key as SPKI PEM alongside the
// kid the extension would register it under (see devicePSSOKID), so tests can
// build registration requests whose kids match their keys.
func mustDevicePSSOKey(t *testing.T) (pemStr, kid string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pemStr = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	kid, err = devicePSSOKID(&key.PublicKey)
	require.NoError(t, err)
	return pemStr, kid
}

func mustECPrivateKeyPEM(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

// TestPSSO_JWKSIncludesSigningAndEncryptionKeys confirms the JWKS publishes both
// the signing key (use:sig, ES256) the device verifies responses with and the
// encryption key (use:enc, ECDH-ES) it encrypts the password to, with distinct
// kids.
func TestPSSO_JWKSIncludesSigningAndEncryptionKeys(t *testing.T) {
	svc, ctx := newPSSOTestService(t, configuredPSSOTestConfig())
	ds := svc.ds.(*mock.Store)

	signKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	encKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signPEM := mustECPrivateKeyPEM(t, signKey)
	encPEM := mustECPrivateKeyPEM(t, encKey)

	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		out := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
		for _, n := range names {
			switch n {
			case fleet.MDMAssetPSSOSigningKey:
				out[n] = fleet.MDMConfigAsset{Name: n, Value: signPEM}
			case fleet.MDMAssetPSSOEncryptionKey:
				out[n] = fleet.MDMConfigAsset{Name: n, Value: encPEM}
			}
		}
		return out, nil
	}

	body, err := svc.PSSOJWKS(ctx)
	require.NoError(t, err)

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(body, &jwks))
	require.Len(t, jwks.Keys, 2)

	byUse := map[string]struct{ alg, kid, crv, kty string }{}
	for _, k := range jwks.Keys {
		assert.Equal(t, "EC", k.Kty)
		assert.Equal(t, "P-256", k.Crv)
		byUse[k.Use] = struct{ alg, kid, crv, kty string }{k.Alg, k.Kid, k.Crv, k.Kty}
	}

	sig, ok := byUse["sig"]
	require.True(t, ok, "signing key (use:sig) must be present")
	assert.Equal(t, "ES256", sig.alg)

	enc, ok := byUse["enc"]
	require.True(t, ok, "encryption key (use:enc) must be present")
	assert.Equal(t, "ECDH-ES", enc.alg)

	assert.NotEqual(t, sig.kid, enc.kid, "signing and encryption keys must have distinct kids")
}

func TestPSSORegisterDevice_RequiresValidToken(t *testing.T) {
	const hostUUID = "A72B07D0-2E08-45CE-9423-1FCAFFAEC390"

	fleetKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	fleetKeyDER, err := x509.MarshalECPrivateKey(fleetKey)
	require.NoError(t, err)
	fleetKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: fleetKeyDER})

	devSigning, signingKID := mustDevicePSSOKey(t)
	devEncryption, encryptionKID := mustDevicePSSOKey(t)

	newSvc := func(t *testing.T) (*Service, context.Context, *mock.Store) {
		svc, ctx := newPSSOTestService(t, configuredPSSOTestConfig())
		ds := svc.ds.(*mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			out := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
			for _, n := range names {
				switch n {
				case fleet.MDMAssetAppleAccountProvisioningIdPClientSecret:
					out[n] = fleet.MDMConfigAsset{Name: n, Value: []byte("client-secret")}
				case fleet.MDMAssetPSSOSigningKey:
					out[n] = fleet.MDMConfigAsset{Name: n, Value: fleetKeyPEM}
				}
			}
			return out, nil
		}
		ds.HostByUUIDFunc = func(_ context.Context, uuid string) (*fleet.Host, error) {
			if uuid != hostUUID {
				return nil, &testNotFoundError{}
			}
			return &fleet.Host{UUID: uuid}, nil
		}
		return svc, ctx, ds
	}

	validReq := func(token string) fleet.PSSODeviceRegistrationRequest {
		return fleet.PSSODeviceRegistrationRequest{
			DeviceUUID:          hostUUID,
			DeviceSigningKey:    devSigning,
			DeviceEncryptionKey: devEncryption,
			SigningKeyID:        signingKID,
			EncryptionKeyID:     encryptionKID,
			RegistrationToken:   token,
		}
	}

	t.Run("valid token registers and derives host from the token subject", func(t *testing.T) {
		svc, ctx, ds := newSvc(t)
		var storedUUID string
		var storedKeys []fleet.PSSOKey
		ds.SetOrUpdatePSSODeviceFunc = func(_ context.Context, uuid string, keys []fleet.PSSOKey) error {
			storedUUID = uuid
			storedKeys = keys
			return nil
		}

		token, err := regtoken.Mint(fleetKey, hostUUID, time.Now())
		require.NoError(t, err)

		require.NoError(t, svc.PSSORegisterDevice(ctx, validReq(token)))
		require.True(t, ds.SetOrUpdatePSSODeviceFuncInvoked)
		require.Equal(t, hostUUID, storedUUID)
		require.Len(t, storedKeys, 2)
		// kids are derived from the keys, not taken from the request labels.
		require.Equal(t, signingKID, storedKeys[0].KID)
		require.Equal(t, encryptionKID, storedKeys[1].KID)
	})

	t.Run("key id that does not match its key is rejected", func(t *testing.T) {
		svc, ctx, ds := newSvc(t)
		token, err := regtoken.Mint(fleetKey, hostUUID, time.Now())
		require.NoError(t, err)

		req := validReq(token)
		// A valid-looking kid that belongs to a different key must be refused, so
		// a device can't claim (and overwrite) another device's key row.
		req.SigningKeyID = encryptionKID
		err = svc.PSSORegisterDevice(ctx, req)
		require.ErrorContains(t, err, "signing key id does not match")
		require.False(t, ds.SetOrUpdatePSSODeviceFuncInvoked)
	})

	t.Run("missing token is rejected", func(t *testing.T) {
		svc, ctx, _ := newSvc(t)
		err := svc.PSSORegisterDevice(ctx, validReq(""))
		require.ErrorContains(t, err, "missing registration token")
	})

	t.Run("token signed by another key is rejected", func(t *testing.T) {
		svc, ctx, ds := newSvc(t)
		otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		token, err := regtoken.Mint(otherKey, hostUUID, time.Now())
		require.NoError(t, err)

		err = svc.PSSORegisterDevice(ctx, validReq(token))
		require.ErrorContains(t, err, "invalid registration token")
		require.False(t, ds.SetOrUpdatePSSODeviceFuncInvoked)
	})

	t.Run("token bound to a non-enrolled host is rejected", func(t *testing.T) {
		svc, ctx, ds := newSvc(t)
		token, err := regtoken.Mint(fleetKey, "11111111-2222-3333-4444-555555555555", time.Now())
		require.NoError(t, err)

		err = svc.PSSORegisterDevice(ctx, validReq(token))
		require.Error(t, err)
		require.ErrorContains(t, err, "no enrolled host")
		require.False(t, ds.SetOrUpdatePSSODeviceFuncInvoked)
	})
}
