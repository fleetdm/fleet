package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

type AccountService interface {
	CreateAccount(ctx context.Context, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.Account, error)
}
