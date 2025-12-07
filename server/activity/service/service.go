// Package service implements the business logic for the activity bounded context.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
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
func (svc *Service) Ping(ctx context.Context) error {
	svc.authz.SkipAuthorization(ctx)
	return svc.store.Ping(ctx)
}

// ListActivities returns activities matching the given options.
func (svc *Service) ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &activity.Activity{}, "read"); err != nil {
		return nil, nil, err
	}
	return svc.store.ListActivities(ctx, opt)
}

// ListHostPastActivities returns past activities for a specific host.
func (svc *Service) ListHostPastActivities(ctx context.Context, hostID uint, opt activity.ListOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	// Note: Host authorization should be done by the caller (handler layer)
	// since it requires access to the host service
	svc.authz.SkipAuthorization(ctx)
	return svc.store.ListHostPastActivities(ctx, hostID, opt)
}

// NewActivity records a new activity in the audit log.
func (svc *Service) NewActivity(ctx context.Context, actor *activity.Actor, details activity.Details, detailsJSON []byte, createdAt time.Time) error {
	// Note: Authorization is handled by the caller, as activities are created
	// as side effects of other authorized operations
	svc.authz.SkipAuthorization(ctx)

	// Mark webhook as processed (caller is responsible for webhook handling)
	ctx = context.WithValue(ctx, activity.WebhookContextKey, true)

	return svc.store.NewActivity(ctx, actor, details, detailsJSON, createdAt)
}

// MarkActivitiesAsStreamed marks activities as streamed to external destinations.
func (svc *Service) MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error {
	// Internal operation, no user authorization needed
	svc.authz.SkipAuthorization(ctx)
	return svc.store.MarkActivitiesAsStreamed(ctx, activityIDs)
}

// CleanupActivitiesAndAssociatedData removes old activities.
func (svc *Service) CleanupActivitiesAndAssociatedData(ctx context.Context, maxCount int, expiryWindowDays int) error {
	// Internal operation (cron job), no user authorization needed
	svc.authz.SkipAuthorization(ctx)
	return svc.store.CleanupActivitiesAndAssociatedData(ctx, maxCount, expiryWindowDays)
}
