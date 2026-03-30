package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

type AuthorizationService interface {
	GetAuthorization(ctx context.Context, enrollment *types.Enrollment, account *types.Account, authorizationID uint) (*types.AuthorizationResponse, error)
}
