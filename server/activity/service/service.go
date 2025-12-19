// Package service implements the business logic for the activity bounded context.
package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/activity"
)

type Service struct {
	authz activity.Authorizer
	store Datastore
}

// NewService creates a new activity service with the given dependencies.
func NewService(authz activity.Authorizer, store Datastore) *Service {
	return &Service{
		authz: authz,
		store: store,
	}
}

// Ping verifies the service is healthy.
func (svc *Service) Ping(ctx context.Context) error {
	svc.authz.SkipAuthorization(ctx)
	return svc.store.Ping(ctx)
}

// ListActivities returns a paginated list of activities for the organization.
func (svc *Service) ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &activity.Activity{}, activity.ActionRead); err != nil {
		return nil, nil, err
	}
	return svc.store.ListActivities(ctx, opt)
}
