// Package service implements the business logic for the activity bounded context.
package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
)

type Service struct {
	authz *authz.Authorizer
	store Datastore
}

// NewService creates a new activity service with the given dependencies.
func NewService(store Datastore) (*Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	return &Service{
		authz: authorizer,
		store: store,
	}, nil
}

// Ping verifies the service is healthy.
// This is a placeholder method for the scaffold phase.
func (svc *Service) Ping(ctx context.Context) error {
	svc.authz.SkipAuthorization(ctx)
	return svc.store.Ping(ctx)
}
