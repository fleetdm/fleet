package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

func (s *Service) CreateAccount(ctx context.Context, enrollmentID uint, jwk jose.JSONWebKey, onlyReturnExisting bool) (*types.Account, error) {
	account := &types.Account{
		EnrollmentID: enrollmentID,
		JSONWebKey:   jwk,
	}
	account, err := s.store.CreateAccount(ctx, account, onlyReturnExisting)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating account in datastore")
	}
	return account, nil
}

func (s *Service) CreateOrder(ctx context.Context, order *types.Order) (*types.Order, error) {
	panic("unimplemented")
}
