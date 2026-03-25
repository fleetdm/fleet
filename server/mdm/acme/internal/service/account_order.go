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
		EnrollmentID: enrollmentID,
		JSONWebKey:   jwk,
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

func (s *Service) CreateOrder(ctx context.Context, order *types.Order) (*types.Order, error) {
	panic("unimplemented")
}
