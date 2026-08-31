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

func (s *Service) CreateNotification(ctx context.Context, notification *api.EndUserNotification) (*api.EndUserNotification, error) {
	if _, kindRegistered := s.kinds[notification.Kind]; !kindRegistered {
		return nil, ctxerr.Errorf(ctx, "no kind registered for end user notification %q", notification.Kind)
	}

	created, err := s.ds.NewEndUserNotification(ctx, notification)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create end user notification")
	}
	return created, nil
}

func (s *Service) notificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*api.EndUserNotification, error) {
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

func (s *Service) RenderNotificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*api.NotificationView, error) {
	notification, err := s.notificationForHost(ctx, hostID, notificationUUID)
	if err != nil {
		return nil, err
	}
	return s.render(ctx, notification)
}

// render is not found rather than an error when the kind is gone, because a
// notification nothing can describe is nothing the end user can be shown.
func (s *Service) render(ctx context.Context, notification *api.EndUserNotification) (*api.NotificationView, error) {
	kind, kindRegistered := s.kinds[notification.Kind]
	if !kindRegistered {
		s.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
		return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: notification.UUID}, "no kind to render notification")
	}

	view, err := kind.Render(ctx, notification)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "notification kind rendering view")
	}
	return view, nil
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

func (s *Service) MarkNotificationActed(ctx context.Context, notificationUUID string) error {
	if err := s.ds.SetEndUserNotificationActed(ctx, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "mark end user notification acted")
	}
	return nil
}
