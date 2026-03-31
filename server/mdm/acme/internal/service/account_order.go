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
	authzURL, err := s.getACMEURLWithBaseURL(ctx, baseURL, enrollment.PathIdentifier, "authorizations", fmt.Sprint(authz.ID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "constructing authorization URL for account")
	}

	return &types.OrderResponse{
		ID:             order.ID,
		Status:         order.Status,
		Expires:        enrollment.NotValidAfter,
		Identifiers:    order.Identifiers,
		Authorizations: []string{authzURL},
		Finalize:       finalizeURL,
		Location:       orderURL,
	}, nil
}
