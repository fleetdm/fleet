package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// PSSOStubIdPClient is a deterministic fleet.PSSOIdPClient used in tests and
// local development. It accepts any non-empty password and returns claims
// derived directly from the username.
type PSSOStubIdPClient struct{}

// ValidatePasswordAndGetClaims accepts any non-empty (username, password)
// pair and returns synthetic OIDC-shaped claims. It exists so the PSSO
// crypto and persistence layers can be exercised end-to-end without
// requiring an upstream OIDC IdP.
func (PSSOStubIdPClient) ValidatePasswordAndGetClaims(_ context.Context, username, password string) (*fleet.PSSOClaims, error) {
	if username == "" || password == "" {
		return nil, &fleet.BadRequestError{Message: "stub IdP requires non-empty username and password"}
	}
	return &fleet.PSSOClaims{
		Subject:           "stub-sub-" + username,
		Email:             username,
		PreferredUsername: username,
		Name:              username,
	}, nil
}
