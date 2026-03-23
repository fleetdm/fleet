package api

import (
	"context"

	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

type AccountService interface {
	CreateAccount(ctx context.Context, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.Account, error)
	AuthenticateNewAccountMessage(ctx context.Context, message *api_http.JWSRequestContainer, request *api_http.CreateNewAccountRequest) error
	AuthenticateMessageFromAccount(ctx context.Context, message *api_http.JWSRequestContainer, request *types.AccountAuthenticatedRequest) error
}
