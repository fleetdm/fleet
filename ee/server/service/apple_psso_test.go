package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func newPSSOTestService(t *testing.T, settings *fleet.PSSOSettings) (*Service, context.Context) {
	t.Helper()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{PSSOSettings: settings}, nil
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

func configuredPSSOSettings() *fleet.PSSOSettings {
	return &fleet.PSSOSettings{
		Enabled:         true,
		IssuerURL:       "https://fleet.example.com",
		IdPTokenURL:     "https://idp.example.com/oauth2/v1/token",
		IdPClientID:     "client-id",
		IdPClientSecret: "client-secret",
	}
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
	notConfigured := []*fleet.PSSOSettings{
		nil,
		{},
		func() *fleet.PSSOSettings { s := configuredPSSOSettings(); s.Enabled = false; return s }(),
		func() *fleet.PSSOSettings { s := configuredPSSOSettings(); s.IssuerURL = ""; return s }(),
		func() *fleet.PSSOSettings { s := configuredPSSOSettings(); s.IdPTokenURL = ""; return s }(),
		func() *fleet.PSSOSettings { s := configuredPSSOSettings(); s.IdPClientID = ""; return s }(),
		func() *fleet.PSSOSettings { s := configuredPSSOSettings(); s.IdPClientSecret = ""; return s }(),
	}

	for _, settings := range notConfigured {
		svc, ctx := newPSSOTestService(t, settings)

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
}

func TestPSSO_NonceIssuedAndConsumedWhenConfigured(t *testing.T) {
	svc, ctx := newPSSOTestService(t, configuredPSSOSettings())
	svc.pssoNonceStore = &memNonceStore{}

	nonce, err := svc.PSSONonce(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, nonce)

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
