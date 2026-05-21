package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

func (s *Service) NewNonce(ctx context.Context, identifier string) error {
	// authentication is via the identifier, that must exist as a valid ACME enrollment
	if _, err := s.authenticateWithACMEEnrollment(ctx, identifier); err != nil {
		return err
	}

	// actual nonce generation happens in the rendering of the response
	return nil
}

func (s *Service) GetDirectory(ctx context.Context, identifier string) (*types.Directory, error) {
	// authentication is via the identifier, that must exist as a valid ACME enrollment
	if _, err := s.authenticateWithACMEEnrollment(ctx, identifier); err != nil {
		return nil, err
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	suffixes := []string{"new_nonce", "new_account", "new_order"}
	urls := make(map[string]string, len(suffixes))
	for _, suffix := range suffixes {
		u, err := s.getACMEURLWithBaseURL(ctx, baseURL, identifier, suffix)
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
