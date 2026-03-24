package service

import (
	"context"
	"fmt"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
)

func (s *Service) NewNonce(ctx context.Context, identifier string) (string, error) {
	// skipauth: No authorization check needed, it is done via path identifier.
	if az, ok := authz_ctx.FromContext(ctx); ok {
		az.SetChecked()
	}

	// authentication is via the identifier, that must exist as a valid ACME enrollment
	if err := s.authenticateWithACMEEnrollment(ctx, identifier); err != nil {
		return "", err
	}

	nonce := types.CreateNonceEncodedForHeader()
	if err := s.nonces.Store(ctx, nonce, redis_nonces_store.DefaultNonceExpiration); err != nil {
		return "", err
	}
	return nonce, nil
}

func (s *Service) GetDirectory(ctx context.Context, identifier string) (*types.Directory, error) {
	// skipauth: No authorization check needed, it is done via path identifier.
	if az, ok := authz_ctx.FromContext(ctx); ok {
		az.SetChecked()
	}

	// authentication is via the identifier, that must exist as a valid ACME enrollment
	if err := s.authenticateWithACMEEnrollment(ctx, identifier); err != nil {
		return nil, err
	}

	appConfig, err := s.providers.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	baseURL := appConfig.ServerSettings.ServerURL
	suffixes := []string{"new_nonce", "new_account", "new_order"}
	urls := make(map[string]string, len(suffixes))
	for _, suffix := range suffixes {
		u, err := commonmdm.ResolveURL(baseURL, fmt.Sprintf("/api/mdm/acme/%s/%s", identifier, suffix), true)
		if err != nil {
			return nil, err
		}
		urls[suffix] = u
	}

	return &types.Directory{
		NewNonce:   urls["new_nonce"],
		NewAccount: urls["new_account"],
		NewOrder:   urls["new_order"],
	}, nil
}
