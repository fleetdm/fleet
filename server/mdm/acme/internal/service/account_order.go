package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

func (s *Service) CreateAccount(ctx context.Context, pathIdentifier string, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.AccountResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	account := &types.Account{
		ACMEEnrollmentID: enrollmentID,
		JSONWebKey:       jwk,
	}
	account, didCreate, err := s.store.CreateAccount(ctx, account, onlyReturnExisting)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating account in datastore")
	}

	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	ordersURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "accounts", fmt.Sprint(account.ID), "orders")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing orders URL for account")
	}
	acctURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "accounts", fmt.Sprint(account.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing account URL for account")
	}

	return &types.AccountResponse{
		CreatedAccount: account,
		DidCreate:      didCreate,
		Status:         "valid", // for now, in our implementation, always valid
		Orders:         ordersURL,
		Location:       acctURL,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, partialOrder *types.Order) (*types.OrderResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	// The "identifiers" passed as part of the newOrder request must be an array with a
	// single member of type "permanent-identifier" matching the serial specified in the
	// acme_enrollment that this enrollment was created for.
	if len(partialOrder.Identifiers) != 1 || partialOrder.Identifiers[0].Type != types.IdentifierTypePermanentIdentifier {
		return nil, types.UnsupportedIdentifierError("A single identifier of type permanent-identifier must be provided in the order request")
	}
	if partialOrder.Identifiers[0].Value != enrollment.HostIdentifier {
		return nil, types.RejectedIdentifierError("The identifier value does not match the host identifier for this enrollment")
	}

	// notBefore and notAfter, which are optional, must not be set because fleet is going
	// to control these and the Apple payload doesn't allow specification of them.
	if partialOrder.NotBefore != nil || partialOrder.NotAfter != nil {
		return nil, types.MalformedError("notBefore and notAfter must not be set in the order request")
	}

	identifiers := []types.Identifier{
		{Type: partialOrder.Identifiers[0].Type, Value: partialOrder.Identifiers[0].Value},
	}
	order := &types.Order{
		ACMEAccountID: account.ID,
		Finalized:     false,
		Identifiers:   identifiers,
		Status:        "pending", // always pending at creation
	}
	authz := &types.Authorization{
		Identifier: identifiers[0],
		Status:     "pending", // always pending at creation
	}
	challenge := &types.Challenge{
		ChallengeType: types.DeviceAttestationChallengeType, // only supported challenge for now
		Token:         types.CreateNonceEncodedForHeader(),
		Status:        "pending", // always pending at creation
	}
	order, err := s.store.CreateOrder(ctx, order, authz, challenge)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating order in datastore")
	}
	return s.createOrderResponse(ctx, enrollment, order, []*types.Authorization{authz})
}

func (s *Service) createOrderResponse(
	ctx context.Context,
	enrollment *types.Enrollment,
	order *types.Order,
	authorizations []*types.Authorization,
) (*types.OrderResponse, error) {
	baseURL, err := s.getACMEBaseURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting base URL")
	}

	orderURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing order URL for account")
	}

	finalizeURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "finalize")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing finalize URL for account")
	}

	var authzURL string
	if len(authorizations) == 1 {
		authzURL, err = s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authorizations[0].ID))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
		}
	}

	var certURL string
	if order.Finalized && order.Status == "valid" {
		certURL, err = s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "orders", fmt.Sprint(order.ID), "certificate")
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "constructing certificate URL for account")
		}
	}

	return &types.OrderResponse{
		ID:             order.ID,
		Status:         order.Status,
		Expires:        enrollment.NotValidAfter,
		Identifiers:    order.Identifiers,
		Authorizations: []string{authzURL},
		Finalize:       finalizeURL,
		Certificate:    certURL,
		Location:       orderURL,
	}, nil
}

func (s *Service) GetOrder(ctx context.Context, enrollment *types.Enrollment, account *types.Account, orderID uint) (*types.OrderResponse, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	order, authorizations, err := s.store.GetOrderByID(ctx, account.ID, orderID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get order from datastore")
	}
	return s.createOrderResponse(ctx, enrollment, order, authorizations)
}

func (s *Service) ListAccountOrders(ctx context.Context, pathIdentifier string, account *types.Account) ([]string, error) {
	// authorization is checked in the endpoint implementation for JWS-protected endpoints

	orderIDs, err := s.store.ListAccountOrderIDs(ctx, account.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing account order IDs from datastore")
	}

	var orderURLs []string
	if len(orderIDs) > 0 {
		baseURL, err := s.getACMEBaseURL(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting base URL")
		}

		orderURLs = make([]string, len(orderIDs))
		for i, orderID := range orderIDs {
			orderURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, pathIdentifier, "orders", fmt.Sprint(orderID))
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "constructing order URL for account")
			}
			orderURLs[i] = orderURL
		}
	}
	return orderURLs, nil
}
