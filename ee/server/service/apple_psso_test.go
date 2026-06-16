package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
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
