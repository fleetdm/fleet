package api

import (
	"context"

	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

type AccountService interface {
	CreateAccount(ctx context.Context, pathIdentifier string, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.AccountResponse, error)
	AuthenticateMessageFromAccount(ctx context.Context, message *api_http.JWSRequestContainer, request types.AccountAuthenticatedRequest) error
	AuthenticateNewAccountMessage(ctx context.Context, message *api_http.JWSRequestContainer, request *api_http.CreateNewAccountRequest) error
	CreateOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, partialOrder *types.Order) (*types.OrderResponse, error)
	FinalizeOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, orderID uint, csr string) (*types.OrderResponse, error)
	GetOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, orderID uint) (*types.OrderResponse, error)
	ListAccountOrders(ctx context.Context, pathIdentifier string, account *types.Account) ([]string, error)
	GetCertificate(ctx context.Context, accountID, orderID uint) (string, error)
}
