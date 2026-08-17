// Package service provides the service implementation for the notifications
// bounded context.
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
)

type Service struct {
	ds          types.Datastore
	scriptQueue notifications.ScriptQueueProvider
	logger      *slog.Logger

	// kinds is populated by RegisterKind before the server starts serving
	// requests, so it's never written to concurrently with a read.
	kinds map[string]api.NotificationKind
}

func NewService(
	ds types.Datastore,
	providers notifications.DataProviders,
	logger *slog.Logger,
) *Service {
	return &Service{
		ds:          ds,
		scriptQueue: providers,
		logger:      logger,
		kinds:       make(map[string]api.NotificationKind),
	}
}

var _ api.Service = (*Service)(nil)

func (s *Service) RegisterKind(kind api.NotificationKind) {
	s.kinds[kind.Name()] = kind
}

func (s *Service) GetNotificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*api.EndUserNotification, error) {
	notification, err := s.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get end user notification")
	}

	// another host's notification is not found rather than forbidden, so the
	// uuid can't be probed
	if notification.HostID != hostID {
		return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: notificationUUID}, "no notification found for this host")
	}

	return notification, nil
}

func (s *Service) NotificationUUIDForExecution(ctx context.Context, executionID string) (string, error) {
	notification, err := s.ds.GetEndUserNotificationByExecutionID(ctx, executionID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get end user notification for execution")
	}
	return notification.UUID, nil
}

func (s *Service) DelayNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time, payload json.RawMessage) error {
	if err := s.ds.DelayEndUserNotification(ctx, notificationUUID, nextAttemptAt, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "delay end user notification")
	}
	return nil
}
