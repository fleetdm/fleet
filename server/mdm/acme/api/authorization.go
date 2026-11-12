package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

// AuthorizationService does not handle normal authentication, but the ACME concept of authorization as part of the protocol.
type AuthorizationService interface {
	GetAuthorization(ctx context.Context, enrollment *types.Enrollment, account *types.Account, authorizationID uint) (*types.AuthorizationResponse, error)
}
