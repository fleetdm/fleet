package bootstrap

import (
	"context"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// NewForUnitTests creates an activity NewActivityService backed by a noop store (no database required).
func NewForUnitTests(
	providers activity.DataProviders,
	webhookSendFn activity.WebhookSendFunc,
	logger *slog.Logger,
) api.NewActivityService {
	return service.NewService(&noopAuthorizer{}, &noopStore{}, providers, webhookSendFn, logger)
}

// noopAuthorizer allows all actions (appropriate for unit tests).
type noopAuthorizer struct{}

func (a *noopAuthorizer) Authorize(_ context.Context, _ platform_authz.AuthzTyper, _ platform_authz.Action) error {
	return nil
}

// noopStore is a datastore that does nothing (appropriate for unit tests that only need webhook behavior).
type noopStore struct{}

func (s *noopStore) ListActivities(_ context.Context, _ types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (s *noopStore) ListHostPastActivities(_ context.Context, _ uint, _ types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	return nil, nil, nil
}

func (s *noopStore) MarkActivitiesAsStreamed(_ context.Context, _ []uint) error {
	return nil
}

func (s *noopStore) NewActivity(_ context.Context, _ *api.User, _ api.ActivityDetails, _ []byte, _ time.Time) error {
	return nil
}

func (s *noopStore) CleanupExpiredActivities(_ context.Context, _ int, _ time.Time) error {
	return nil
}
